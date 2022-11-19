// Copyright 2015 The go-ETX Authors
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

package etx

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ETX/go-ETX"
	"github.com/ETX/go-ETX/accounts"
	"github.com/ETX/go-ETX/common"
	"github.com/ETX/go-ETX/consensus"
	"github.com/ETX/go-ETX/core"
	"github.com/ETX/go-ETX/core/bloombits"
	"github.com/ETX/go-ETX/core/rawdb"
	"github.com/ETX/go-ETX/core/state"
	"github.com/ETX/go-ETX/core/txpool"
	"github.com/ETX/go-ETX/core/types"
	"github.com/ETX/go-ETX/core/vm"
	"github.com/ETX/go-ETX/etx/gasprice"
	"github.com/ETX/go-ETX/etx/tracers"
	"github.com/ETX/go-ETX/etxdb"
	"github.com/ETX/go-ETX/event"
	"github.com/ETX/go-ETX/miner"
	"github.com/ETX/go-ETX/params"
	"github.com/ETX/go-ETX/rpc"
)

// etxAPIBackend implements etxapi.Backend for full nodes
type etxAPIBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	etx                 *ETX
	gpo                 *gasprice.Oracle
}

// ChainConfig returns the active chain configuration.
func (b *etxAPIBackend) ChainConfig() *params.ChainConfig {
	return b.etx.blockchain.Config()
}

func (b *etxAPIBackend) CurrentBlock() *types.Block {
	return b.etx.blockchain.CurrentBlock()
}

func (b *etxAPIBackend) Setxead(number uint64) {
	b.etx.handler.downloader.Cancel()
	b.etx.blockchain.Setxead(number)
}

func (b *etxAPIBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.etx.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.etx.blockchain.CurrentBlock().Header(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		block := b.etx.blockchain.CurrentFinalizedBlock()
		if block != nil {
			return block.Header(), nil
		}
		return nil, errors.New("finalized block not found")
	}
	if number == rpc.SafeBlockNumber {
		block := b.etx.blockchain.CurrentSafeBlock()
		if block != nil {
			return block.Header(), nil
		}
		return nil, errors.New("safe block not found")
	}
	return b.etx.blockchain.GetxeaderByNumber(uint64(number)), nil
}

func (b *etxAPIBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.etx.blockchain.GetxeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.etx.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return header, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *etxAPIBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.etx.blockchain.GetxeaderByHash(hash), nil
}

func (b *etxAPIBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if number == rpc.PendingBlockNumber {
		block := b.etx.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if number == rpc.LatestBlockNumber {
		return b.etx.blockchain.CurrentBlock(), nil
	}
	if number == rpc.FinalizedBlockNumber {
		return b.etx.blockchain.CurrentFinalizedBlock(), nil
	}
	if number == rpc.SafeBlockNumber {
		return b.etx.blockchain.CurrentSafeBlock(), nil
	}
	return b.etx.blockchain.GetBlockByNumber(uint64(number)), nil
}

func (b *etxAPIBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.etx.blockchain.GetBlockByHash(hash), nil
}

func (b *etxAPIBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.etx.blockchain.GetxeaderByHash(hash)
		if header == nil {
			return nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.etx.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		block := b.etx.blockchain.GetBlock(hash, header.Number.Uint64())
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *etxAPIBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return b.etx.miner.PendingBlockAndReceipts()
}

func (b *etxAPIBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if number == rpc.PendingBlockNumber {
		block, state := b.etx.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	stateDb, err := b.etx.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *etxAPIBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, nil, err
		}
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.etx.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		stateDb, err := b.etx.BlockChain().StateAt(header.Root)
		return stateDb, header, err
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *etxAPIBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return b.etx.blockchain.GetReceiptsByHash(hash), nil
}

func (b *etxAPIBackend) GetLogs(ctx context.Context, hash common.Hash, number uint64) ([][]*types.Log, error) {
	return rawdb.ReadLogs(b.etx.chainDb, hash, number, b.ChainConfig()), nil
}

func (b *etxAPIBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	if header := b.etx.blockchain.GetxeaderByHash(hash); header != nil {
		return b.etx.blockchain.GetTd(hash, header.Number.Uint64())
	}
	return nil
}

func (b *etxAPIBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	if vmConfig == nil {
		vmConfig = b.etx.blockchain.GetVMConfig()
	}
	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.etx.BlockChain(), nil)
	return vm.NewEVM(context, txContext, state, b.etx.blockchain.Config(), *vmConfig), state.Error, nil
}

func (b *etxAPIBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.etx.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *etxAPIBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.etx.miner.SubscribePendingLogs(ch)
}

func (b *etxAPIBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.etx.BlockChain().SubscribeChainEvent(ch)
}

func (b *etxAPIBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.etx.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *etxAPIBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.etx.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *etxAPIBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.etx.BlockChain().SubscribeLogsEvent(ch)
}

func (b *etxAPIBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.etx.txPool.AddLocal(signedTx)
}

func (b *etxAPIBackend) GetPoolTransactions() (types.Transactions, error) {
	pending := b.etx.txPool.Pending(false)
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *etxAPIBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.etx.txPool.Get(hash)
}

func (b *etxAPIBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	tx, blockHash, blockNumber, index := rawdb.ReadTransaction(b.etx.ChainDb(), txHash)
	return tx, blockHash, blockNumber, index, nil
}

func (b *etxAPIBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.etx.txPool.Nonce(addr), nil
}

func (b *etxAPIBackend) Stats() (pending int, queued int) {
	return b.etx.txPool.Stats()
}

func (b *etxAPIBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.etx.TxPool().Content()
}

func (b *etxAPIBackend) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	return b.etx.TxPool().ContentFrom(addr)
}

func (b *etxAPIBackend) TxPool() *txpool.TxPool {
	return b.etx.TxPool()
}

func (b *etxAPIBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.etx.TxPool().SubscribeNewTxsEvent(ch)
}

func (b *etxAPIBackend) SyncProgress() ETX.SyncProgress {
	return b.etx.Downloader().Progress()
}

func (b *etxAPIBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *etxAPIBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *etxAPIBackend) ChainDb() etxdb.Database {
	return b.etx.ChainDb()
}

func (b *etxAPIBackend) EventMux() *event.TypeMux {
	return b.etx.EventMux()
}

func (b *etxAPIBackend) AccountManager() *accounts.Manager {
	return b.etx.AccountManager()
}

func (b *etxAPIBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *etxAPIBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *etxAPIBackend) RPCGasCap() uint64 {
	return b.etx.config.RPCGasCap
}

func (b *etxAPIBackend) RPCEVMTimeout() time.Duration {
	return b.etx.config.RPCEVMTimeout
}

func (b *etxAPIBackend) RPCTxFeeCap() float64 {
	return b.etx.config.RPCTxFeeCap
}

func (b *etxAPIBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.etx.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *etxAPIBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.etx.bloomRequests)
	}
}

func (b *etxAPIBackend) Engine() consensus.Engine {
	return b.etx.engine
}

func (b *etxAPIBackend) CurrentHeader() *types.Header {
	return b.etx.blockchain.CurrentHeader()
}

func (b *etxAPIBackend) Miner() *miner.Miner {
	return b.etx.Miner()
}

func (b *etxAPIBackend) StartMining(threads int) error {
	return b.etx.StartMining(threads)
}

func (b *etxAPIBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error) {
	return b.etx.StateAtBlock(block, reexec, base, readOnly, preferDisk)
}

func (b *etxAPIBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	return b.etx.stateAtTransaction(block, txIndex, reexec)
}
