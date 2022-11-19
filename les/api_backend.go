// Copyright 2016 The go-ETX Authors
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

package les

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
	"github.com/ETX/go-ETX/core/types"
	"github.com/ETX/go-ETX/core/vm"
	"github.com/ETX/go-ETX/etx/gasprice"
	"github.com/ETX/go-ETX/etx/tracers"
	"github.com/ETX/go-ETX/etxdb"
	"github.com/ETX/go-ETX/event"
	"github.com/ETX/go-ETX/light"
	"github.com/ETX/go-ETX/params"
	"github.com/ETX/go-ETX/rpc"
)

type LesApiBackend struct {
	extRPCEnabled       bool
	allowUnprotectedTxs bool
	etx                 *LightETX
	gpo                 *gasprice.Oracle
}

func (b *LesApiBackend) ChainConfig() *params.ChainConfig {
	return b.etx.chainConfig
}

func (b *LesApiBackend) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(b.etx.BlockChain().CurrentHeader())
}

func (b *LesApiBackend) Setxead(number uint64) {
	b.etx.handler.downloader.Cancel()
	b.etx.blockchain.Setxead(number)
}

func (b *LesApiBackend) HeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Header, error) {
	// Return the latest current as the pending one since there
	// is no pending notion in the light client. TODO(rjl493456442)
	// unify the behavior of `HeaderByNumber` and `PendingBlockAndReceipts`.
	if number == rpc.PendingBlockNumber {
		return b.etx.blockchain.CurrentHeader(), nil
	}
	if number == rpc.LatestBlockNumber {
		return b.etx.blockchain.CurrentHeader(), nil
	}
	return b.etx.blockchain.GetxeaderByNumberOdr(ctx, uint64(number))
}

func (b *LesApiBackend) HeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.HeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header, err := b.HeaderByHash(ctx, hash)
		if err != nil {
			return nil, err
		}
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

func (b *LesApiBackend) HeaderByHash(ctx context.Context, hash common.Hash) (*types.Header, error) {
	return b.etx.blockchain.GetxeaderByHash(hash), nil
}

func (b *LesApiBackend) BlockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	header, err := b.HeaderByNumber(ctx, number)
	if header == nil || err != nil {
		return nil, err
	}
	return b.BlockByHash(ctx, header.Hash())
}

func (b *LesApiBackend) BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	return b.etx.blockchain.GetBlockByHash(ctx, hash)
}

func (b *LesApiBackend) BlockByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*types.Block, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.BlockByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err := b.BlockByHash(ctx, hash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return nil, errors.New("header found, but block body is missing")
		}
		if blockNrOrHash.RequireCanonical && b.etx.blockchain.GetCanonicalHash(block.NumberU64()) != hash {
			return nil, errors.New("hash is not currently canonical")
		}
		return block, nil
	}
	return nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *LesApiBackend) PendingBlockAndReceipts() (*types.Block, types.Receipts) {
	return nil, nil
}

func (b *LesApiBackend) StateAndHeaderByNumber(ctx context.Context, number rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	header, err := b.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, nil, err
	}
	if header == nil {
		return nil, nil, errors.New("header not found")
	}
	return light.NewState(ctx, header, b.etx.odr), header, nil
}

func (b *LesApiBackend) StateAndHeaderByNumberOrHash(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*state.StateDB, *types.Header, error) {
	if blockNr, ok := blockNrOrHash.Number(); ok {
		return b.StateAndHeaderByNumber(ctx, blockNr)
	}
	if hash, ok := blockNrOrHash.Hash(); ok {
		header := b.etx.blockchain.GetxeaderByHash(hash)
		if header == nil {
			return nil, nil, errors.New("header for hash not found")
		}
		if blockNrOrHash.RequireCanonical && b.etx.blockchain.GetCanonicalHash(header.Number.Uint64()) != hash {
			return nil, nil, errors.New("hash is not currently canonical")
		}
		return light.NewState(ctx, header, b.etx.odr), header, nil
	}
	return nil, nil, errors.New("invalid arguments; neither block nor hash specified")
}

func (b *LesApiBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	if number := rawdb.ReadHeaderNumber(b.etx.chainDb, hash); number != nil {
		return light.GetBlockReceipts(ctx, b.etx.odr, hash, *number)
	}
	return nil, nil
}

func (b *LesApiBackend) GetLogs(ctx context.Context, hash common.Hash, number uint64) ([][]*types.Log, error) {
	return light.GetBlockLogs(ctx, b.etx.odr, hash, number)
}

func (b *LesApiBackend) GetTd(ctx context.Context, hash common.Hash) *big.Int {
	if number := rawdb.ReadHeaderNumber(b.etx.chainDb, hash); number != nil {
		return b.etx.blockchain.GetTdOdr(ctx, hash, *number)
	}
	return nil
}

func (b *LesApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmConfig *vm.Config) (*vm.EVM, func() error, error) {
	if vmConfig == nil {
		vmConfig = new(vm.Config)
	}
	txContext := core.NewEVMTxContext(msg)
	context := core.NewEVMBlockContext(header, b.etx.blockchain, nil)
	return vm.NewEVM(context, txContext, state, b.etx.chainConfig, *vmConfig), state.Error, nil
}

func (b *LesApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.etx.txPool.Add(ctx, signedTx)
}

func (b *LesApiBackend) RemoveTx(txHash common.Hash) {
	b.etx.txPool.RemoveTx(txHash)
}

func (b *LesApiBackend) GetPoolTransactions() (types.Transactions, error) {
	return b.etx.txPool.GetTransactions()
}

func (b *LesApiBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	return b.etx.txPool.GetTransaction(txHash)
}

func (b *LesApiBackend) GetTransaction(ctx context.Context, txHash common.Hash) (*types.Transaction, common.Hash, uint64, uint64, error) {
	return light.GetTransaction(ctx, b.etx.odr, txHash)
}

func (b *LesApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.etx.txPool.GetNonce(ctx, addr)
}

func (b *LesApiBackend) Stats() (pending int, queued int) {
	return b.etx.txPool.Stats(), 0
}

func (b *LesApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.etx.txPool.Content()
}

func (b *LesApiBackend) TxPoolContentFrom(addr common.Address) (types.Transactions, types.Transactions) {
	return b.etx.txPool.ContentFrom(addr)
}

func (b *LesApiBackend) SubscribeNewTxsEvent(ch chan<- core.NewTxsEvent) event.Subscription {
	return b.etx.txPool.SubscribeNewTxsEvent(ch)
}

func (b *LesApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.etx.blockchain.SubscribeChainEvent(ch)
}

func (b *LesApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.etx.blockchain.SubscribeChainHeadEvent(ch)
}

func (b *LesApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.etx.blockchain.SubscribeChainSideEvent(ch)
}

func (b *LesApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.etx.blockchain.SubscribeLogsEvent(ch)
}

func (b *LesApiBackend) SubscribePendingLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		<-quit
		return nil
	})
}

func (b *LesApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.etx.blockchain.SubscribeRemovedLogsEvent(ch)
}

func (b *LesApiBackend) SyncProgress() ETX.SyncProgress {
	return b.etx.Downloader().Progress()
}

func (b *LesApiBackend) ProtocolVersion() int {
	return b.etx.LesVersion() + 10000
}

func (b *LesApiBackend) SuggestGasTipCap(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestTipCap(ctx)
}

func (b *LesApiBackend) FeeHistory(ctx context.Context, blockCount int, lastBlock rpc.BlockNumber, rewardPercentiles []float64) (firstBlock *big.Int, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	return b.gpo.FeeHistory(ctx, blockCount, lastBlock, rewardPercentiles)
}

func (b *LesApiBackend) ChainDb() etxdb.Database {
	return b.etx.chainDb
}

func (b *LesApiBackend) AccountManager() *accounts.Manager {
	return b.etx.accountManager
}

func (b *LesApiBackend) ExtRPCEnabled() bool {
	return b.extRPCEnabled
}

func (b *LesApiBackend) UnprotectedAllowed() bool {
	return b.allowUnprotectedTxs
}

func (b *LesApiBackend) RPCGasCap() uint64 {
	return b.etx.config.RPCGasCap
}

func (b *LesApiBackend) RPCEVMTimeout() time.Duration {
	return b.etx.config.RPCEVMTimeout
}

func (b *LesApiBackend) RPCTxFeeCap() float64 {
	return b.etx.config.RPCTxFeeCap
}

func (b *LesApiBackend) BloomStatus() (uint64, uint64) {
	if b.etx.bloomIndexer == nil {
		return 0, 0
	}
	sections, _, _ := b.etx.bloomIndexer.Sections()
	return params.BloomBitsBlocksClient, sections
}

func (b *LesApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.etx.bloomRequests)
	}
}

func (b *LesApiBackend) Engine() consensus.Engine {
	return b.etx.engine
}

func (b *LesApiBackend) CurrentHeader() *types.Header {
	return b.etx.blockchain.CurrentHeader()
}

func (b *LesApiBackend) StateAtBlock(ctx context.Context, block *types.Block, reexec uint64, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error) {
	return b.etx.stateAtBlock(ctx, block, reexec)
}

func (b *LesApiBackend) StateAtTransaction(ctx context.Context, block *types.Block, txIndex int, reexec uint64) (core.Message, vm.BlockContext, *state.StateDB, tracers.StateReleaseFunc, error) {
	return b.etx.stateAtTransaction(ctx, block, txIndex, reexec)
}
