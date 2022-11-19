// Copyright 2014 The go-ETX Authors
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

// Package etx implements the ETX protocol.
package etx

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ETX/go-ETX/accounts"
	"github.com/ETX/go-ETX/common"
	"github.com/ETX/go-ETX/common/hexutil"
	"github.com/ETX/go-ETX/consensus"
	"github.com/ETX/go-ETX/consensus/beacon"
	"github.com/ETX/go-ETX/consensus/clique"
	"github.com/ETX/go-ETX/core"
	"github.com/ETX/go-ETX/core/bloombits"
	"github.com/ETX/go-ETX/core/rawdb"
	"github.com/ETX/go-ETX/core/state/pruner"
	"github.com/ETX/go-ETX/core/txpool"
	"github.com/ETX/go-ETX/core/types"
	"github.com/ETX/go-ETX/core/vm"
	"github.com/ETX/go-ETX/etx/downloader"
	"github.com/ETX/go-ETX/etx/etxconfig"
	"github.com/ETX/go-ETX/etx/gasprice"
	"github.com/ETX/go-ETX/etx/protocols/etx"
	"github.com/ETX/go-ETX/etx/protocols/snap"
	"github.com/ETX/go-ETX/etxdb"
	"github.com/ETX/go-ETX/event"
	"github.com/ETX/go-ETX/internal/etxapi"
	"github.com/ETX/go-ETX/internal/shutdowncheck"
	"github.com/ETX/go-ETX/log"
	"github.com/ETX/go-ETX/miner"
	"github.com/ETX/go-ETX/node"
	"github.com/ETX/go-ETX/p2p"
	"github.com/ETX/go-ETX/p2p/dnsdisc"
	"github.com/ETX/go-ETX/p2p/enode"
	"github.com/ETX/go-ETX/params"
	"github.com/ETX/go-ETX/rlp"
	"github.com/ETX/go-ETX/rpc"
)

// Config contains the configuration options of the etx protocol.
// Deprecated: use etxconfig.Config instead.
type Config = etxconfig.Config

// ETX implements the ETX full node service.
type ETX struct {
	config *etxconfig.Config

	// Handlers
	txPool             *txpool.TxPool
	blockchain         *core.BlockChain
	handler            *handler
	etxDialCandidates  enode.Iterator
	snapDialCandidates enode.Iterator
	merger             *consensus.Merger

	// DB interfaces
	chainDb etxdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests     chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer      *core.ChainIndexer             // Bloom indexer operating during block imports
	closeBloomHandler chan struct{}

	APIBackend *etxAPIBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	etxerbase common.Address

	networkID     uint64
	netRPCService *etxapi.NetAPI

	p2pServer *p2p.Server

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etxerbase)

	shutdownTracker *shutdowncheck.ShutdownTracker // Tracks if and when the node has shutdown ungracefully
}

// New creates a new ETX object (including the
// initialisation of the common ETX object)
func New(stack *node.Node, config *etxconfig.Config) (*ETX, error) {
	// Ensure configuration values are compatible and sane
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run etx.ETX in light sync mode, use les.LightETX")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	if config.Miner.GasPrice == nil || config.Miner.GasPrice.Cmp(common.Big0) <= 0 {
		log.Warn("Sanitizing invalid miner gas price", "provided", config.Miner.GasPrice, "updated", etxconfig.Defaults.Miner.GasPrice)
		config.Miner.GasPrice = new(big.Int).Set(etxconfig.Defaults.Miner.GasPrice)
	}
	if config.NoPruning && config.TrieDirtyCache > 0 {
		if config.SnapshotCache > 0 {
			config.TrieCleanCache += config.TrieDirtyCache * 3 / 5
			config.SnapshotCache += config.TrieDirtyCache * 2 / 5
		} else {
			config.TrieCleanCache += config.TrieDirtyCache
		}
		config.TrieDirtyCache = 0
	}
	log.Info("Allocated trie memory caches", "clean", common.StorageSize(config.TrieCleanCache)*1024*1024, "dirty", common.StorageSize(config.TrieDirtyCache)*1024*1024)

	// Assemble the ETX object
	chainDb, err := stack.OpenDatabaseWithFreezer("chaindata", config.DatabaseCache, config.DatabaseHandles, config.DatabaseFreezer, "etx/db/chaindata/", false)
	if err != nil {
		return nil, err
	}
	if err := pruner.RecoverPruning(stack.ResolvePath(""), chainDb, stack.ResolvePath(config.TrieCleanCacheJournal)); err != nil {
		log.Error("Failed to recover state", "error", err)
	}
	// Transfer mining-related config to the etxash config.
	etxashConfig := config.etxash
	etxashConfig.NotifyFull = config.Miner.NotifyFull
	cliqueConfig, err := core.LoadCliqueConfig(chainDb, config.Genesis)
	if err != nil {
		return nil, err
	}
	engine := etxconfig.CreateConsensusEngine(stack, &etxashConfig, cliqueConfig, config.Miner.Notify, config.Miner.Noverify, chainDb)

	etx := &ETX{
		config:            config,
		merger:            consensus.NewMerger(chainDb),
		chainDb:           chainDb,
		eventMux:          stack.EventMux(),
		accountManager:    stack.AccountManager(),
		engine:            engine,
		closeBloomHandler: make(chan struct{}),
		networkID:         config.NetworkId,
		gasPrice:          config.Miner.GasPrice,
		etxerbase:         config.Miner.etxerbase,
		bloomRequests:     make(chan chan *bloombits.Retrieval),
		bloomIndexer:      core.NewBloomIndexer(chainDb, params.BloomBitsBlocks, params.BloomConfirms),
		p2pServer:         stack.Server(),
		shutdownTracker:   shutdowncheck.NewShutdownTracker(chainDb),
	}

	bcVersion := rawdb.ReadDatabaseVersion(chainDb)
	var dbVer = "<nil>"
	if bcVersion != nil {
		dbVer = fmt.Sprintf("%d", *bcVersion)
	}
	log.Info("Initialising ETX protocol", "network", config.NetworkId, "dbversion", dbVer)

	if !config.SkipBcVersionCheck {
		if bcVersion != nil && *bcVersion > core.BlockChainVersion {
			return nil, fmt.Errorf("database version is v%d, Getx %s only supports v%d", *bcVersion, params.VersionWithMeta, core.BlockChainVersion)
		} else if bcVersion == nil || *bcVersion < core.BlockChainVersion {
			if bcVersion != nil { // only print warning on upgrade, not on init
				log.Warn("Upgrade blockchain database version", "from", dbVer, "to", core.BlockChainVersion)
			}
			rawdb.WriteDatabaseVersion(chainDb, core.BlockChainVersion)
		}
	}
	var (
		vmConfig = vm.Config{
			EnablePreimageRecording: config.EnablePreimageRecording,
		}
		cacheConfig = &core.CacheConfig{
			TrieCleanLimit:      config.TrieCleanCache,
			TrieCleanJournal:    stack.ResolvePath(config.TrieCleanCacheJournal),
			TrieCleanRejournal:  config.TrieCleanCacheRejournal,
			TrieCleanNoPrefetch: config.NoPrefetch,
			TrieDirtyLimit:      config.TrieDirtyCache,
			TrieDirtyDisabled:   config.NoPruning,
			TrieTimeLimit:       config.TrieTimeout,
			SnapshotLimit:       config.SnapshotCache,
			Preimages:           config.Preimages,
		}
	)
	// Override the chain config with provided settings.
	var overrides core.ChainOverrides
	if config.OverrideTerminalTotalDifficulty != nil {
		overrides.OverrideTerminalTotalDifficulty = config.OverrideTerminalTotalDifficulty
	}
	if config.OverrideTerminalTotalDifficultyPassed != nil {
		overrides.OverrideTerminalTotalDifficultyPassed = config.OverrideTerminalTotalDifficultyPassed
	}
	etx.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, config.Genesis, &overrides, etx.engine, vmConfig, etx.shouldPreserve, &config.TxLookupLimit)
	if err != nil {
		return nil, err
	}
	etx.bloomIndexer.Start(etx.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = stack.ResolvePath(config.TxPool.Journal)
	}
	etx.txPool = txpool.NewTxPool(config.TxPool, etx.blockchain.Config(), etx.blockchain)

	// Permit the downloader to use the trie cache allowance during fast sync
	cacheLimit := cacheConfig.TrieCleanLimit + cacheConfig.TrieDirtyLimit + cacheConfig.SnapshotLimit
	checkpoint := config.Checkpoint
	if checkpoint == nil {
		checkpoint = params.TrustedCheckpoints[etx.blockchain.Genesis().Hash()]
	}
	if etx.handler, err = newHandler(&handlerConfig{
		Database:       chainDb,
		Chain:          etx.blockchain,
		TxPool:         etx.txPool,
		Merger:         etx.merger,
		Network:        config.NetworkId,
		Sync:           config.SyncMode,
		BloomCache:     uint64(cacheLimit),
		EventMux:       etx.eventMux,
		Checkpoint:     checkpoint,
		RequiredBlocks: config.RequiredBlocks,
	}); err != nil {
		return nil, err
	}

	etx.miner = miner.New(etx, &config.Miner, etx.blockchain.Config(), etx.EventMux(), etx.engine, etx.isLocalBlock)
	etx.miner.SetExtra(makeExtraData(config.Miner.ExtraData))

	etx.APIBackend = &etxAPIBackend{stack.Config().ExtRPCEnabled(), stack.Config().AllowUnprotectedTxs, etx, nil}
	if etx.APIBackend.allowUnprotectedTxs {
		log.Info("Unprotected transactions allowed")
	}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.Miner.GasPrice
	}
	etx.APIBackend.gpo = gasprice.NewOracle(etx.APIBackend, gpoParams)

	// Setup DNS discovery iterators.
	dnsclient := dnsdisc.NewClient(dnsdisc.Config{})
	etx.etxDialCandidates, err = dnsclient.NewIterator(etx.config.etxDiscoveryURLs...)
	if err != nil {
		return nil, err
	}
	etx.snapDialCandidates, err = dnsclient.NewIterator(etx.config.SnapDiscoveryURLs...)
	if err != nil {
		return nil, err
	}

	// Start the RPC service
	etx.netRPCService = etxapi.NewNetAPI(etx.p2pServer, config.NetworkId)

	// Register the backend on the node
	stack.RegisterAPIs(etx.APIs())
	stack.RegisterProtocols(etx.Protocols())
	stack.RegisterLifecycle(etx)

	// Successful startup; push a marker and check previous unclean shutdowns.
	etx.shutdownTracker.MarkStartup()

	return etx, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {
		// create default extradata
		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"getx",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

// APIs return the collection of RPC services the ETX package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *ETX) APIs() []rpc.API {
	apis := etxapi.GetAPIs(s.APIBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "etx",
			Service:   NewETXAPI(s),
		}, {
			Namespace: "miner",
			Service:   NewMinerAPI(s),
		}, {
			Namespace: "etx",
			Service:   downloader.NewDownloaderAPI(s.handler.downloader, s.eventMux),
		}, {
			Namespace: "admin",
			Service:   NewAdminAPI(s),
		}, {
			Namespace: "debug",
			Service:   NewDebugAPI(s),
		}, {
			Namespace: "net",
			Service:   s.netRPCService,
		},
	}...)
}

func (s *ETX) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *ETX) etxerbase() (eb common.Address, err error) {
	s.lock.RLock()
	etxerbase := s.etxerbase
	s.lock.RUnlock()

	if etxerbase != (common.Address{}) {
		return etxerbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			etxerbase := accounts[0].Address

			s.lock.Lock()
			s.etxerbase = etxerbase
			s.lock.Unlock()

			log.Info("etxerbase automatically configured", "address", etxerbase)
			return etxerbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("etxerbase must be explicitly specified")
}

// isLocalBlock checks whetxer the specified block is mined
// by local miner accounts.
//
// We regard two types of accounts as local miner account: etxerbase
// and accounts specified via `txpool.locals` flag.
func (s *ETX) isLocalBlock(header *types.Header) bool {
	author, err := s.engine.Author(header)
	if err != nil {
		log.Warn("Failed to retrieve block author", "number", header.Number.Uint64(), "hash", header.Hash(), "err", err)
		return false
	}
	// Check whetxer the given address is etxerbase.
	s.lock.RLock()
	etxerbase := s.etxerbase
	s.lock.RUnlock()
	if author == etxerbase {
		return true
	}
	// Check whetxer the given address is specified by `txpool.local`
	// CLI flag.
	for _, account := range s.config.TxPool.Locals {
		if account == author {
			return true
		}
	}
	return false
}

// shouldPreserve checks whetxer we should preserve the given block
// during the chain reorg depending on whetxer the author of block
// is a local account.
func (s *ETX) shouldPreserve(header *types.Header) bool {
	// The reason we need to disable the self-reorg preserving for clique
	// is it can be probable to introduce a deadlock.
	//
	// e.g. If there are 7 available signers
	//
	// r1   A
	// r2     B
	// r3       C
	// r4         D
	// r5   A      [X] F G
	// r6    [X]
	//
	// In the round5, the inturn signer E is offline, so the worst case
	// is A, F and G sign the block of round5 and reject the block of opponents
	// and in the round6, the last available signer B is offline, the whole
	// network is stuck.
	if _, ok := s.engine.(*clique.Clique); ok {
		return false
	}
	return s.isLocalBlock(header)
}

// Setetxerbase sets the mining reward address.
func (s *ETX) Setetxerbase(etxerbase common.Address) {
	s.lock.Lock()
	s.etxerbase = etxerbase
	s.lock.Unlock()

	s.miner.Setetxerbase(etxerbase)
}

// StartMining starts the miner with the given number of CPU threads. If mining
// is already running, this metxod adjust the number of threads allowed to use
// and updates the minimum price required by the transaction pool.
func (s *ETX) StartMining(threads int) error {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		log.Info("Updated mining threads", "threads", threads)
		if threads == 0 {
			threads = -1 // Disable the miner from within
		}
		th.SetThreads(threads)
	}
	// If the miner was not running, initialize it
	if !s.IsMining() {
		// Propagate the initial price point to the transaction pool
		s.lock.RLock()
		price := s.gasPrice
		s.lock.RUnlock()
		s.txPool.SetGasPrice(price)

		// Configure the local mining address
		eb, err := s.etxerbase()
		if err != nil {
			log.Error("Cannot start mining without etxerbase", "err", err)
			return fmt.Errorf("etxerbase missing: %v", err)
		}
		var cli *clique.Clique
		if c, ok := s.engine.(*clique.Clique); ok {
			cli = c
		} else if cl, ok := s.engine.(*beacon.Beacon); ok {
			if c, ok := cl.InnerEngine().(*clique.Clique); ok {
				cli = c
			}
		}
		if cli != nil {
			wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
			if wallet == nil || err != nil {
				log.Error("etxerbase account unavailable locally", "err", err)
				return fmt.Errorf("signer missing: %v", err)
			}
			cli.Authorize(eb, wallet.SignData)
		}
		// If mining is started, we can disable the transaction rejection mechanism
		// introduced to speed sync times.
		atomic.StoreUint32(&s.handler.acceptTxs, 1)

		go s.miner.Start(eb)
	}
	return nil
}

// StopMining terminates the miner, both at the consensus engine level as well as
// at the block creation level.
func (s *ETX) StopMining() {
	// Update the thread count within the consensus engine
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := s.engine.(threaded); ok {
		th.SetThreads(-1)
	}
	// Stop the block creating itself
	s.miner.Stop()
}

func (s *ETX) IsMining() bool      { return s.miner.Mining() }
func (s *ETX) Miner() *miner.Miner { return s.miner }

func (s *ETX) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *ETX) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *ETX) TxPool() *txpool.TxPool             { return s.txPool }
func (s *ETX) EventMux() *event.TypeMux           { return s.eventMux }
func (s *ETX) Engine() consensus.Engine           { return s.engine }
func (s *ETX) ChainDb() etxdb.Database            { return s.chainDb }
func (s *ETX) IsListening() bool                  { return true } // Always listening
func (s *ETX) Downloader() *downloader.Downloader { return s.handler.downloader }
func (s *ETX) Synced() bool                       { return atomic.LoadUint32(&s.handler.acceptTxs) == 1 }
func (s *ETX) SetSynced()                         { atomic.StoreUint32(&s.handler.acceptTxs, 1) }
func (s *ETX) ArchiveMode() bool                  { return s.config.NoPruning }
func (s *ETX) BloomIndexer() *core.ChainIndexer   { return s.bloomIndexer }
func (s *ETX) Merger() *consensus.Merger          { return s.merger }
func (s *ETX) SyncMode() downloader.SyncMode {
	mode, _ := s.handler.chainSync.modeAndLocalHead()
	return mode
}

// Protocols returns all the currently configured
// network protocols to start.
func (s *ETX) Protocols() []p2p.Protocol {
	protos := etx.MakeProtocols((*etxHandler)(s.handler), s.networkID, s.etxDialCandidates)
	if s.config.SnapshotCache > 0 {
		protos = append(protos, snap.MakeProtocols((*snapHandler)(s.handler), s.snapDialCandidates)...)
	}
	return protos
}

// Start implements node.Lifecycle, starting all internal goroutines needed by the
// ETX protocol implementation.
func (s *ETX) Start() error {
	etx.StartENRUpdater(s.blockchain, s.p2pServer.LocalNode())

	// Start the bloom bits servicing goroutines
	s.startBloomHandlers(params.BloomBitsBlocks)

	// Regularly update shutdown marker
	s.shutdownTracker.Start()

	// Figure out a max peers count based on the server limits
	maxPeers := s.p2pServer.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= s.p2pServer.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, s.p2pServer.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.handler.Start(maxPeers)
	return nil
}

// Stop implements node.Lifecycle, terminating all internal goroutines used by the
// ETX protocol.
func (s *ETX) Stop() error {
	// Stop all the peer-related stuff first.
	s.etxDialCandidates.Close()
	s.snapDialCandidates.Close()
	s.handler.Stop()

	// Then stop everything else.
	s.bloomIndexer.Close()
	close(s.closeBloomHandler)
	s.txPool.Stop()
	s.miner.Close()
	s.blockchain.Stop()
	s.engine.Close()

	// Clean shutdown marker as the last thing before closing db
	s.shutdownTracker.Stop()

	s.chainDb.Close()
	s.eventMux.Stop()

	return nil
}
