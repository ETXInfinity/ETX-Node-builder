// Code generated by github.com/fjl/gencodec. DO NOT EDIT.

package etxconfig

import (
	"math/big"
	"time"

	"github.com/ETX/go-ETX/common"
	"github.com/ETX/go-ETX/consensus/etxash"
	"github.com/ETX/go-ETX/core"
	"github.com/ETX/go-ETX/core/txpool"
	"github.com/ETX/go-ETX/core/types"
	"github.com/ETX/go-ETX/etx/downloader"
	"github.com/ETX/go-ETX/etx/gasprice"
	"github.com/ETX/go-ETX/miner"
	"github.com/ETX/go-ETX/params"
)

// MarshalTOML marshals as TOML.
func (c Config) MarshalTOML() (interface{}, error) {
	type Config struct {
		Genesis                               *core.Genesis `toml:",omitempty"`
		NetworkId                             uint64
		SyncMode                              downloader.SyncMode
		etxDiscoveryURLs                      []string
		SnapDiscoveryURLs                     []string
		NoPruning                             bool
		NoPrefetch                            bool
		TxLookupLimit                         uint64                 `toml:",omitempty"`
		RequiredBlocks                        map[uint64]common.Hash `toml:"-"`
		LightServ                             int                    `toml:",omitempty"`
		LightIngress                          int                    `toml:",omitempty"`
		LightEgress                           int                    `toml:",omitempty"`
		LightPeers                            int                    `toml:",omitempty"`
		LightNoPrune                          bool                   `toml:",omitempty"`
		LightNoSyncServe                      bool                   `toml:",omitempty"`
		SyncFromCheckpoint                    bool                   `toml:",omitempty"`
		UltraLightServers                     []string               `toml:",omitempty"`
		UltraLightFraction                    int                    `toml:",omitempty"`
		UltraLightOnlyAnnounce                bool                   `toml:",omitempty"`
		SkipBcVersionCheck                    bool                   `toml:"-"`
		DatabaseHandles                       int                    `toml:"-"`
		DatabaseCache                         int
		DatabaseFreezer                       string
		TrieCleanCache                        int
		TrieCleanCacheJournal                 string        `toml:",omitempty"`
		TrieCleanCacheRejournal               time.Duration `toml:",omitempty"`
		TrieDirtyCache                        int
		TrieTimeout                           time.Duration
		SnapshotCache                         int
		Preimages                             bool
		FilterLogCacheSize                    int
		Miner                                 miner.Config
		etxash                                etxash.Config
		TxPool                                txpool.Config
		GPO                                   gasprice.Config
		EnablePreimageRecording               bool
		DocRoot                               string `toml:"-"`
		RPCGasCap                             uint64
		RPCEVMTimeout                         time.Duration
		RPCTxFeeCap                           float64
		Checkpoint                            *params.TrustedCheckpoint      `toml:",omitempty"`
		CheckpointOracle                      *params.CheckpointOracleConfig `toml:",omitempty"`
		OverrideTerminalTotalDifficulty       *big.Int                       `toml:",omitempty"`
		OverrideTerminalTotalDifficultyPassed *bool                          `toml:",omitempty"`
		FullSyncTarget                        *types.Block
	}
	var enc Config
	enc.Genesis = c.Genesis
	enc.NetworkId = c.NetworkId
	enc.SyncMode = c.SyncMode
	enc.etxDiscoveryURLs = c.etxDiscoveryURLs
	enc.SnapDiscoveryURLs = c.SnapDiscoveryURLs
	enc.NoPruning = c.NoPruning
	enc.NoPrefetch = c.NoPrefetch
	enc.TxLookupLimit = c.TxLookupLimit
	enc.RequiredBlocks = c.RequiredBlocks
	enc.LightServ = c.LightServ
	enc.LightIngress = c.LightIngress
	enc.LightEgress = c.LightEgress
	enc.LightPeers = c.LightPeers
	enc.LightNoPrune = c.LightNoPrune
	enc.LightNoSyncServe = c.LightNoSyncServe
	enc.SyncFromCheckpoint = c.SyncFromCheckpoint
	enc.UltraLightServers = c.UltraLightServers
	enc.UltraLightFraction = c.UltraLightFraction
	enc.UltraLightOnlyAnnounce = c.UltraLightOnlyAnnounce
	enc.SkipBcVersionCheck = c.SkipBcVersionCheck
	enc.DatabaseHandles = c.DatabaseHandles
	enc.DatabaseCache = c.DatabaseCache
	enc.DatabaseFreezer = c.DatabaseFreezer
	enc.TrieCleanCache = c.TrieCleanCache
	enc.TrieCleanCacheJournal = c.TrieCleanCacheJournal
	enc.TrieCleanCacheRejournal = c.TrieCleanCacheRejournal
	enc.TrieDirtyCache = c.TrieDirtyCache
	enc.TrieTimeout = c.TrieTimeout
	enc.SnapshotCache = c.SnapshotCache
	enc.Preimages = c.Preimages
	enc.FilterLogCacheSize = c.FilterLogCacheSize
	enc.Miner = c.Miner
	enc.etxash = c.etxash
	enc.TxPool = c.TxPool
	enc.GPO = c.GPO
	enc.EnablePreimageRecording = c.EnablePreimageRecording
	enc.DocRoot = c.DocRoot
	enc.RPCGasCap = c.RPCGasCap
	enc.RPCEVMTimeout = c.RPCEVMTimeout
	enc.RPCTxFeeCap = c.RPCTxFeeCap
	enc.Checkpoint = c.Checkpoint
	enc.CheckpointOracle = c.CheckpointOracle
	enc.OverrideTerminalTotalDifficulty = c.OverrideTerminalTotalDifficulty
	enc.OverrideTerminalTotalDifficultyPassed = c.OverrideTerminalTotalDifficultyPassed
	enc.FullSyncTarget = c.SyncTarget
	return &enc, nil
}

// UnmarshalTOML unmarshals from TOML.
func (c *Config) UnmarshalTOML(unmarshal func(interface{}) error) error {
	type Config struct {
		Genesis                               *core.Genesis `toml:",omitempty"`
		NetworkId                             *uint64
		SyncMode                              *downloader.SyncMode
		etxDiscoveryURLs                      []string
		SnapDiscoveryURLs                     []string
		NoPruning                             *bool
		NoPrefetch                            *bool
		TxLookupLimit                         *uint64                `toml:",omitempty"`
		RequiredBlocks                        map[uint64]common.Hash `toml:"-"`
		LightServ                             *int                   `toml:",omitempty"`
		LightIngress                          *int                   `toml:",omitempty"`
		LightEgress                           *int                   `toml:",omitempty"`
		LightPeers                            *int                   `toml:",omitempty"`
		LightNoPrune                          *bool                  `toml:",omitempty"`
		LightNoSyncServe                      *bool                  `toml:",omitempty"`
		SyncFromCheckpoint                    *bool                  `toml:",omitempty"`
		UltraLightServers                     []string               `toml:",omitempty"`
		UltraLightFraction                    *int                   `toml:",omitempty"`
		UltraLightOnlyAnnounce                *bool                  `toml:",omitempty"`
		SkipBcVersionCheck                    *bool                  `toml:"-"`
		DatabaseHandles                       *int                   `toml:"-"`
		DatabaseCache                         *int
		DatabaseFreezer                       *string
		TrieCleanCache                        *int
		TrieCleanCacheJournal                 *string        `toml:",omitempty"`
		TrieCleanCacheRejournal               *time.Duration `toml:",omitempty"`
		TrieDirtyCache                        *int
		TrieTimeout                           *time.Duration
		SnapshotCache                         *int
		Preimages                             *bool
		FilterLogCacheSize                    *int
		Miner                                 *miner.Config
		etxash                                *etxash.Config
		TxPool                                *txpool.Config
		GPO                                   *gasprice.Config
		EnablePreimageRecording               *bool
		DocRoot                               *string `toml:"-"`
		RPCGasCap                             *uint64
		RPCEVMTimeout                         *time.Duration
		RPCTxFeeCap                           *float64
		Checkpoint                            *params.TrustedCheckpoint      `toml:",omitempty"`
		CheckpointOracle                      *params.CheckpointOracleConfig `toml:",omitempty"`
		OverrideTerminalTotalDifficulty       *big.Int                       `toml:",omitempty"`
		OverrideTerminalTotalDifficultyPassed *bool                          `toml:",omitempty"`
		FullSyncTarget                        *types.Block
	}
	var dec Config
	if err := unmarshal(&dec); err != nil {
		return err
	}
	if dec.Genesis != nil {
		c.Genesis = dec.Genesis
	}
	if dec.NetworkId != nil {
		c.NetworkId = *dec.NetworkId
	}
	if dec.SyncMode != nil {
		c.SyncMode = *dec.SyncMode
	}
	if dec.etxDiscoveryURLs != nil {
		c.etxDiscoveryURLs = dec.etxDiscoveryURLs
	}
	if dec.SnapDiscoveryURLs != nil {
		c.SnapDiscoveryURLs = dec.SnapDiscoveryURLs
	}
	if dec.NoPruning != nil {
		c.NoPruning = *dec.NoPruning
	}
	if dec.NoPrefetch != nil {
		c.NoPrefetch = *dec.NoPrefetch
	}
	if dec.TxLookupLimit != nil {
		c.TxLookupLimit = *dec.TxLookupLimit
	}
	if dec.RequiredBlocks != nil {
		c.RequiredBlocks = dec.RequiredBlocks
	}
	if dec.LightServ != nil {
		c.LightServ = *dec.LightServ
	}
	if dec.LightIngress != nil {
		c.LightIngress = *dec.LightIngress
	}
	if dec.LightEgress != nil {
		c.LightEgress = *dec.LightEgress
	}
	if dec.LightPeers != nil {
		c.LightPeers = *dec.LightPeers
	}
	if dec.LightNoPrune != nil {
		c.LightNoPrune = *dec.LightNoPrune
	}
	if dec.LightNoSyncServe != nil {
		c.LightNoSyncServe = *dec.LightNoSyncServe
	}
	if dec.SyncFromCheckpoint != nil {
		c.SyncFromCheckpoint = *dec.SyncFromCheckpoint
	}
	if dec.UltraLightServers != nil {
		c.UltraLightServers = dec.UltraLightServers
	}
	if dec.UltraLightFraction != nil {
		c.UltraLightFraction = *dec.UltraLightFraction
	}
	if dec.UltraLightOnlyAnnounce != nil {
		c.UltraLightOnlyAnnounce = *dec.UltraLightOnlyAnnounce
	}
	if dec.SkipBcVersionCheck != nil {
		c.SkipBcVersionCheck = *dec.SkipBcVersionCheck
	}
	if dec.DatabaseHandles != nil {
		c.DatabaseHandles = *dec.DatabaseHandles
	}
	if dec.DatabaseCache != nil {
		c.DatabaseCache = *dec.DatabaseCache
	}
	if dec.DatabaseFreezer != nil {
		c.DatabaseFreezer = *dec.DatabaseFreezer
	}
	if dec.TrieCleanCache != nil {
		c.TrieCleanCache = *dec.TrieCleanCache
	}
	if dec.TrieCleanCacheJournal != nil {
		c.TrieCleanCacheJournal = *dec.TrieCleanCacheJournal
	}
	if dec.TrieCleanCacheRejournal != nil {
		c.TrieCleanCacheRejournal = *dec.TrieCleanCacheRejournal
	}
	if dec.TrieDirtyCache != nil {
		c.TrieDirtyCache = *dec.TrieDirtyCache
	}
	if dec.TrieTimeout != nil {
		c.TrieTimeout = *dec.TrieTimeout
	}
	if dec.SnapshotCache != nil {
		c.SnapshotCache = *dec.SnapshotCache
	}
	if dec.Preimages != nil {
		c.Preimages = *dec.Preimages
	}
	if dec.FilterLogCacheSize != nil {
		c.FilterLogCacheSize = *dec.FilterLogCacheSize
	}
	if dec.Miner != nil {
		c.Miner = *dec.Miner
	}
	if dec.etxash != nil {
		c.etxash = *dec.etxash
	}
	if dec.TxPool != nil {
		c.TxPool = *dec.TxPool
	}
	if dec.GPO != nil {
		c.GPO = *dec.GPO
	}
	if dec.EnablePreimageRecording != nil {
		c.EnablePreimageRecording = *dec.EnablePreimageRecording
	}
	if dec.DocRoot != nil {
		c.DocRoot = *dec.DocRoot
	}
	if dec.RPCGasCap != nil {
		c.RPCGasCap = *dec.RPCGasCap
	}
	if dec.RPCEVMTimeout != nil {
		c.RPCEVMTimeout = *dec.RPCEVMTimeout
	}
	if dec.RPCTxFeeCap != nil {
		c.RPCTxFeeCap = *dec.RPCTxFeeCap
	}
	if dec.Checkpoint != nil {
		c.Checkpoint = dec.Checkpoint
	}
	if dec.CheckpointOracle != nil {
		c.CheckpointOracle = dec.CheckpointOracle
	}
	if dec.OverrideTerminalTotalDifficulty != nil {
		c.OverrideTerminalTotalDifficulty = dec.OverrideTerminalTotalDifficulty
	}
	if dec.OverrideTerminalTotalDifficultyPassed != nil {
		c.OverrideTerminalTotalDifficultyPassed = dec.OverrideTerminalTotalDifficultyPassed
	}
	if dec.FullSyncTarget != nil {
		c.SyncTarget = dec.FullSyncTarget
	}
	return nil
}