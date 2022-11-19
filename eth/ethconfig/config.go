// Copyright 2021 The go-ETX Authors
// This file is part of the go-ETX library.
//
// The go-ETX library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ETX library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ETX library. If not, see <http://www.gnu.org/licenses/>.

// Package etxconfig contains the configuration of the etx and LES protocols.
package etxconfig

import (
	"math/big"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ETX/go-ETX/common"
	"github.com/ETX/go-ETX/consensus"
	"github.com/ETX/go-ETX/consensus/beacon"
	"github.com/ETX/go-ETX/consensus/clique"
	"github.com/ETX/go-ETX/consensus/etxash"
	"github.com/ETX/go-ETX/core"
	"github.com/ETX/go-ETX/core/txpool"
	"github.com/ETX/go-ETX/core/types"
	"github.com/ETX/go-ETX/etx/downloader"
	"github.com/ETX/go-ETX/etx/gasprice"
	"github.com/ETX/go-ETX/etxdb"
	"github.com/ETX/go-ETX/log"
	"github.com/ETX/go-ETX/miner"
	"github.com/ETX/go-ETX/node"
	"github.com/ETX/go-ETX/params"
)

// FullNodeGPO contains default gasprice oracle settings for full node.
var FullNodeGPO = gasprice.Config{
	Blocks:           20,
	Percentile:       60,
	MaxHeaderHistory: 1024,
	MaxBlockHistory:  1024,
	MaxPrice:         gasprice.DefaultMaxPrice,
	IgnorePrice:      gasprice.DefaultIgnorePrice,
}

// LightClientGPO contains default gasprice oracle settings for light client.
var LightClientGPO = gasprice.Config{
	Blocks:           2,
	Percentile:       60,
	MaxHeaderHistory: 300,
	MaxBlockHistory:  5,
	MaxPrice:         gasprice.DefaultMaxPrice,
	IgnorePrice:      gasprice.DefaultIgnorePrice,
}

// Defaults contains default settings for use on the ETX main net.
var Defaults = Config{
	SyncMode: downloader.SnapSync,
	etxash: etxash.Config{
		CacheDir:         "etxash",
		CachesInMem:      2,
		CachesOnDisk:     3,
		CachesLockMmap:   false,
		DatasetsInMem:    1,
		DatasetsOnDisk:   2,
		DatasetsLockMmap: false,
	},
	NetworkId:               1,
	TxLookupLimit:           2350000,
	LightPeers:              100,
	UltraLightFraction:      75,
	DatabaseCache:           512,
	TrieCleanCache:          154,
	TrieCleanCacheJournal:   "triecache",
	TrieCleanCacheRejournal: 60 * time.Minute,
	TrieDirtyCache:          256,
	TrieTimeout:             60 * time.Minute,
	SnapshotCache:           102,
	FilterLogCacheSize:      32,
	Miner:                   miner.DefaultConfig,
	TxPool:                  txpool.DefaultConfig,
	RPCGasCap:               50000000,
	RPCEVMTimeout:           5 * time.Second,
	GPO:                     FullNodeGPO,
	RPCTxFeeCap:             1, // 1 etxer
}

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		if user, err := user.Current(); err == nil {
			home = user.HomeDir
		}
	}
	if runtime.GOOS == "darwin" {
		Defaults.etxash.DatasetDir = filepath.Join(home, "Library", "etxash")
	} else if runtime.GOOS == "windows" {
		localappdata := os.Getenv("LOCALAPPDATA")
		if localappdata != "" {
			Defaults.etxash.DatasetDir = filepath.Join(localappdata, "etxash")
		} else {
			Defaults.etxash.DatasetDir = filepath.Join(home, "AppData", "Local", "etxash")
		}
	} else {
		Defaults.etxash.DatasetDir = filepath.Join(home, ".etxash")
	}
}

//go:generate go run github.com/fjl/gencodec -type Config -formats toml -out gen_config.go

// Config contains configuration options for of the etx and LES protocols.
type Config struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the ETX main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	NetworkId uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode

	// This can be set to list of enrtree:// URLs which will be queried for
	// for nodes to connect to.
	etxDiscoveryURLs  []string
	SnapDiscoveryURLs []string

	NoPruning  bool // Whetxer to disable pruning and flush everything to disk
	NoPrefetch bool // Whetxer to disable prefetching and only load state on demand

	TxLookupLimit uint64 `toml:",omitempty"` // The maximum number of blocks from head whose tx indices are reserved.

	// RequiredBlocks is a set of block number -> hash mappings which must be in the
	// canonical chain of all remote peers. Setting the option makes getx verify the
	// presence of these blocks for every new peer connection.
	RequiredBlocks map[uint64]common.Hash `toml:"-"`

	// Light client options
	LightServ          int  `toml:",omitempty"` // Maximum percentage of time allowed for serving LES requests
	LightIngress       int  `toml:",omitempty"` // Incoming bandwidth limit for light servers
	LightEgress        int  `toml:",omitempty"` // Outgoing bandwidth limit for light servers
	LightPeers         int  `toml:",omitempty"` // Maximum number of LES client peers
	LightNoPrune       bool `toml:",omitempty"` // Whetxer to disable light chain pruning
	LightNoSyncServe   bool `toml:",omitempty"` // Whetxer to serve light clients before syncing
	SyncFromCheckpoint bool `toml:",omitempty"` // Whetxer to sync the header chain from the configured checkpoint

	// Ultra Light client options
	UltraLightServers      []string `toml:",omitempty"` // List of trusted ultra light servers
	UltraLightFraction     int      `toml:",omitempty"` // Percentage of trusted servers to accept an announcement
	UltraLightOnlyAnnounce bool     `toml:",omitempty"` // Whetxer to only announce headers, or also serve them

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	DatabaseFreezer    string

	TrieCleanCache          int
	TrieCleanCacheJournal   string        `toml:",omitempty"` // Disk journal directory for trie cache to survive node restarts
	TrieCleanCacheRejournal time.Duration `toml:",omitempty"` // Time interval to regenerate the journal for clean cache
	TrieDirtyCache          int
	TrieTimeout             time.Duration
	SnapshotCache           int
	Preimages               bool

	// This is the number of blocks for which logs will be cached in the filter system.
	FilterLogCacheSize int

	// Mining options
	Miner miner.Config

	// etxash options
	etxash etxash.Config

	// Transaction pool options
	TxPool txpool.Config

	// Gas Price Oracle options
	GPO gasprice.Config

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options
	DocRoot string `toml:"-"`

	// RPCGasCap is the global gas cap for etx-call variants.
	RPCGasCap uint64

	// RPCEVMTimeout is the global timeout for etx-call.
	RPCEVMTimeout time.Duration

	// RPCTxFeeCap is the global transaction fee(price * gaslimit) cap for
	// send-transaction variants. The unit is etxer.
	RPCTxFeeCap float64

	// Checkpoint is a hardcoded checkpoint which can be nil.
	Checkpoint *params.TrustedCheckpoint `toml:",omitempty"`

	// CheckpointOracle is the configuration for checkpoint oracle.
	CheckpointOracle *params.CheckpointOracleConfig `toml:",omitempty"`

	// OverrideTerminalTotalDifficulty (TODO: remove after the fork)
	OverrideTerminalTotalDifficulty *big.Int `toml:",omitempty"`

	// OverrideTerminalTotalDifficultyPassed (TODO: remove after the fork)
	OverrideTerminalTotalDifficultyPassed *bool `toml:",omitempty"`

	// SyncTarget defines the target block of sync. It's only used for
	// development purposes.
	SyncTarget *types.Block
}

// CreateConsensusEngine creates a consensus engine for the given chain configuration.
func CreateConsensusEngine(stack *node.Node, etxashConfig *etxash.Config, cliqueConfig *params.CliqueConfig, notify []string, noverify bool, db etxdb.Database) consensus.Engine {
	// If proof-of-authority is requested, set it up
	var engine consensus.Engine
	if cliqueConfig != nil {
		engine = clique.New(cliqueConfig, db)
	} else {
		switch etxashConfig.PowMode {
		case etxash.ModeFake:
			log.Warn("etxash used in fake mode")
		case etxash.ModeTest:
			log.Warn("etxash used in test mode")
		case etxash.ModeShared:
			log.Warn("etxash used in shared mode")
		}
		engine = etxash.New(etxash.Config{
			PowMode:          etxashConfig.PowMode,
			CacheDir:         stack.ResolvePath(etxashConfig.CacheDir),
			CachesInMem:      etxashConfig.CachesInMem,
			CachesOnDisk:     etxashConfig.CachesOnDisk,
			CachesLockMmap:   etxashConfig.CachesLockMmap,
			DatasetDir:       etxashConfig.DatasetDir,
			DatasetsInMem:    etxashConfig.DatasetsInMem,
			DatasetsOnDisk:   etxashConfig.DatasetsOnDisk,
			DatasetsLockMmap: etxashConfig.DatasetsLockMmap,
			NotifyFull:       etxashConfig.NotifyFull,
		}, notify, noverify)
		engine.(*etxash.etxash).SetThreads(-1) // Disable CPU mining
	}
	return beacon.New(engine)
}
