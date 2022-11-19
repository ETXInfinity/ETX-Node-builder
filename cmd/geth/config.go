// Copyright 2017 The go-ETX Authors
// This file is part of go-ETX.
//
// go-ETX is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ETX is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ETX. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"unicode"

	"github.com/urfave/cli/v2"

	"github.com/ETX/go-ETX/accounts/external"
	"github.com/ETX/go-ETX/accounts/keystore"
	"github.com/ETX/go-ETX/accounts/scwallet"
	"github.com/ETX/go-ETX/accounts/usbwallet"
	"github.com/ETX/go-ETX/cmd/utils"
	"github.com/ETX/go-ETX/core/rawdb"
	"github.com/ETX/go-ETX/etx/etxconfig"
	"github.com/ETX/go-ETX/internal/etxapi"
	"github.com/ETX/go-ETX/internal/flags"
	"github.com/ETX/go-ETX/internal/version"
	"github.com/ETX/go-ETX/log"
	"github.com/ETX/go-ETX/metrics"
	"github.com/ETX/go-ETX/node"
	"github.com/ETX/go-ETX/params"
	"github.com/naoina/toml"
)

var (
	dumpConfigCommand = &cli.Command{
		Action:      dumpConfig,
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       flags.Merge(nodeFlags, rpcFlags),
		Description: `The dumpconfig command shows configuration values.`,
	}

	configFileFlag = &cli.StringFlag{
		Name:     "config",
		Usage:    "TOML configuration file",
		Category: flags.etxCategory,
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		id := fmt.Sprintf("%s.%s", rt.String(), field)
		if deprecated(id) {
			log.Warn("Config field is deprecated and won't have an effect", "name", id)
			return nil
		}
		var link string
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://godoc.org/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type etxstatsConfig struct {
	URL string `toml:",omitempty"`
}

type getxConfig struct {
	etx      etxconfig.Config
	Node     node.Config
	etxstats etxstatsConfig
	Metrics  metrics.Config
}

func loadConfig(file string, cfg *getxConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

func defaultNodeConfig() node.Config {
	git, _ := version.VCS()
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(git.Commit, git.Date)
	cfg.HTTPModules = append(cfg.HTTPModules, "etx")
	cfg.WSModules = append(cfg.WSModules, "etx")
	cfg.IPCPath = "getx.ipc"
	return cfg
}

// makeConfigNode loads getx configuration and creates a blank node instance.
func makeConfigNode(ctx *cli.Context) (*node.Node, getxConfig) {
	// Load defaults.
	cfg := getxConfig{
		etx:     etxconfig.Defaults,
		Node:    defaultNodeConfig(),
		Metrics: metrics.DefaultConfig,
	}

	// Load config file.
	if file := ctx.String(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	// Node doesn't by default populate account manager backends
	if err := setAccountManagerBackends(stack); err != nil {
		utils.Fatalf("Failed to set account manager backends: %v", err)
	}

	utils.SetetxConfig(ctx, stack, &cfg.etx)
	if ctx.IsSet(utils.etxStatsURLFlag.Name) {
		cfg.etxstats.URL = ctx.String(utils.etxStatsURLFlag.Name)
	}
	applyMetricConfig(ctx, &cfg)

	return stack, cfg
}

// makeFullNode loads getx configuration and creates the ETX backend.
func makeFullNode(ctx *cli.Context) (*node.Node, etxapi.Backend) {
	stack, cfg := makeConfigNode(ctx)
	if ctx.IsSet(utils.OverrideTerminalTotalDifficulty.Name) {
		cfg.etx.OverrideTerminalTotalDifficulty = flags.GlobalBig(ctx, utils.OverrideTerminalTotalDifficulty.Name)
	}
	if ctx.IsSet(utils.OverrideTerminalTotalDifficultyPassed.Name) {
		override := ctx.Bool(utils.OverrideTerminalTotalDifficultyPassed.Name)
		cfg.etx.OverrideTerminalTotalDifficultyPassed = &override
	}

	backend, etx := utils.RegisteretxService(stack, &cfg.etx)

	// Warn users to migrate if they have a legacy freezer format.
	if etx != nil && !ctx.IsSet(utils.IgnoreLegacyReceiptsFlag.Name) {
		firstIdx := uint64(0)
		// Hack to speed up check for mainnet because we know
		// the first non-empty block.
		ghash := rawdb.ReadCanonicalHash(etx.ChainDb(), 0)
		if cfg.etx.NetworkId == 1 && ghash == params.MainnetGenesisHash {
			firstIdx = 46147
		}
		isLegacy, firstLegacy, err := dbHasLegacyReceipts(etx.ChainDb(), firstIdx)
		if err != nil {
			log.Error("Failed to check db for legacy receipts", "err", err)
		} else if isLegacy {
			stack.Close()
			log.Error("Database has receipts with a legacy format", "firstLegacy", firstLegacy)
			utils.Fatalf("Aborting. Please run `getx db freezer-migrate`.")
		}
	}

	// Configure log filter RPC API.
	filterSystem := utils.RegisterFilterAPI(stack, backend, &cfg.etx)

	// Configure GraphQL if requested.
	if ctx.IsSet(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, backend, filterSystem, &cfg.Node)
	}

	// Add the ETX Stats daemon if requested.
	if cfg.etxstats.URL != "" {
		utils.RegisteretxStatsService(stack, backend, cfg.etxstats.URL)
	}
	return stack, backend
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx *cli.Context) error {
	_, cfg := makeConfigNode(ctx)
	comment := ""

	if cfg.etx.Genesis != nil {
		cfg.etx.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}

	dump := os.Stdout
	if ctx.NArg() > 0 {
		dump, err = os.OpenFile(ctx.Args().Get(0), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer dump.Close()
	}
	dump.WriteString(comment)
	dump.Write(out)

	return nil
}

func applyMetricConfig(ctx *cli.Context, cfg *getxConfig) {
	if ctx.IsSet(utils.MetricsEnabledFlag.Name) {
		cfg.Metrics.Enabled = ctx.Bool(utils.MetricsEnabledFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnabledExpensiveFlag.Name) {
		cfg.Metrics.EnabledExpensive = ctx.Bool(utils.MetricsEnabledExpensiveFlag.Name)
	}
	if ctx.IsSet(utils.MetricsHTTPFlag.Name) {
		cfg.Metrics.HTTP = ctx.String(utils.MetricsHTTPFlag.Name)
	}
	if ctx.IsSet(utils.MetricsPortFlag.Name) {
		cfg.Metrics.Port = ctx.Int(utils.MetricsPortFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBFlag.Name) {
		cfg.Metrics.EnableInfluxDB = ctx.Bool(utils.MetricsEnableInfluxDBFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBEndpointFlag.Name) {
		cfg.Metrics.InfluxDBEndpoint = ctx.String(utils.MetricsInfluxDBEndpointFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBDatabaseFlag.Name) {
		cfg.Metrics.InfluxDBDatabase = ctx.String(utils.MetricsInfluxDBDatabaseFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBUsernameFlag.Name) {
		cfg.Metrics.InfluxDBUsername = ctx.String(utils.MetricsInfluxDBUsernameFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBPasswordFlag.Name) {
		cfg.Metrics.InfluxDBPassword = ctx.String(utils.MetricsInfluxDBPasswordFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBTagsFlag.Name) {
		cfg.Metrics.InfluxDBTags = ctx.String(utils.MetricsInfluxDBTagsFlag.Name)
	}
	if ctx.IsSet(utils.MetricsEnableInfluxDBV2Flag.Name) {
		cfg.Metrics.EnableInfluxDBV2 = ctx.Bool(utils.MetricsEnableInfluxDBV2Flag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBTokenFlag.Name) {
		cfg.Metrics.InfluxDBToken = ctx.String(utils.MetricsInfluxDBTokenFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBBucketFlag.Name) {
		cfg.Metrics.InfluxDBBucket = ctx.String(utils.MetricsInfluxDBBucketFlag.Name)
	}
	if ctx.IsSet(utils.MetricsInfluxDBOrganizationFlag.Name) {
		cfg.Metrics.InfluxDBOrganization = ctx.String(utils.MetricsInfluxDBOrganizationFlag.Name)
	}
}

func deprecated(field string) bool {
	switch field {
	case "etxconfig.Config.EVMInterpreter":
		return true
	case "etxconfig.Config.EWASMInterpreter":
		return true
	default:
		return false
	}
}

func setAccountManagerBackends(stack *node.Node) error {
	conf := stack.Config()
	am := stack.AccountManager()
	keydir := stack.KeyStoreDir()
	scryptN := keystore.StandardScryptN
	scryptP := keystore.StandardScryptP
	if conf.UseLightweightKDF {
		scryptN = keystore.LightScryptN
		scryptP = keystore.LightScryptP
	}

	// Assemble the supported backends
	if len(conf.ExternalSigner) > 0 {
		log.Info("Using external signer", "url", conf.ExternalSigner)
		if extapi, err := external.NewExternalBackend(conf.ExternalSigner); err == nil {
			am.AddBackend(extapi)
			return nil
		} else {
			return fmt.Errorf("error connecting to external signer: %v", err)
		}
	}

	// For now, we're using EITHER external signer OR local signers.
	// If/when we implement some form of lockfile for USB and keystore wallets,
	// we can have both, but it's very confusing for the user to see the same
	// accounts in both externally and locally, plus very racey.
	am.AddBackend(keystore.NewKeyStore(keydir, scryptN, scryptP))
	if conf.USB {
		// Start a USB hub for Ledger hardware wallets
		if ledgerhub, err := usbwallet.NewLedgerHub(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start Ledger hub, disabling: %v", err))
		} else {
			am.AddBackend(ledgerhub)
		}
		// Start a USB hub for Trezor hardware wallets (HID version)
		if trezorhub, err := usbwallet.NewTrezorHubWithHID(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start HID Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
		// Start a USB hub for Trezor hardware wallets (WebUSB version)
		if trezorhub, err := usbwallet.NewTrezorHubWithWebUSB(); err != nil {
			log.Warn(fmt.Sprintf("Failed to start WebUSB Trezor hub, disabling: %v", err))
		} else {
			am.AddBackend(trezorhub)
		}
	}
	if len(conf.SmartCardDaemonPath) > 0 {
		// Start a smart card hub
		if schub, err := scwallet.NewHub(conf.SmartCardDaemonPath, scwallet.Scheme, keydir); err != nil {
			log.Warn(fmt.Sprintf("Failed to start smart card hub, disabling: %v", err))
		} else {
			am.AddBackend(schub)
		}
	}

	return nil
}
