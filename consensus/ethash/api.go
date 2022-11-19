// Copyright 2018 The go-ETX Authors
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

package etxash

import (
	"errors"

	"github.com/ETX/go-ETX/common"
	"github.com/ETX/go-ETX/common/hexutil"
	"github.com/ETX/go-ETX/core/types"
)

var erretxashStopped = errors.New("etxash stopped")

// API exposes etxash related metxods for the RPC interface.
type API struct {
	etxash *etxash
}

// GetWork returns a work package for external miner.
//
// The work package consists of 3 strings:
//
//	result[0] - 32 bytes hex encoded current block header pow-hash
//	result[1] - 32 bytes hex encoded seed hash used for DAG
//	result[2] - 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
//	result[3] - hex encoded block number
func (api *API) GetWork() ([4]string, error) {
	if api.etxash.remote == nil {
		return [4]string{}, errors.New("not supported")
	}

	var (
		workCh = make(chan [4]string, 1)
		errc   = make(chan error, 1)
	)
	select {
	case api.etxash.remote.fetchWorkCh <- &sealWork{errc: errc, res: workCh}:
	case <-api.etxash.remote.exitCh:
		return [4]string{}, erretxashStopped
	}
	select {
	case work := <-workCh:
		return work, nil
	case err := <-errc:
		return [4]string{}, err
	}
}

// SubmitWork can be used by external miner to submit their POW solution.
// It returns an indication if the work was accepted.
// Note either an invalid solution, a stale work a non-existent work will return false.
func (api *API) SubmitWork(nonce types.BlockNonce, hash, digest common.Hash) bool {
	if api.etxash.remote == nil {
		return false
	}

	var errc = make(chan error, 1)
	select {
	case api.etxash.remote.submitWorkCh <- &mineResult{
		nonce:     nonce,
		mixDigest: digest,
		hash:      hash,
		errc:      errc,
	}:
	case <-api.etxash.remote.exitCh:
		return false
	}
	err := <-errc
	return err == nil
}

// SubmitHashrate can be used for remote miners to submit their hash rate.
// This enables the node to report the combined hash rate of all miners
// which submit work through this node.
//
// It accepts the miner hash rate and an identifier which must be unique
// between nodes.
func (api *API) SubmitHashrate(rate hexutil.Uint64, id common.Hash) bool {
	if api.etxash.remote == nil {
		return false
	}

	var done = make(chan struct{}, 1)
	select {
	case api.etxash.remote.submitRateCh <- &hashrate{done: done, rate: uint64(rate), id: id}:
	case <-api.etxash.remote.exitCh:
		return false
	}

	// Block until hash rate submitted successfully.
	<-done
	return true
}

// Getxashrate returns the current hashrate for local CPU miner and remote miner.
func (api *API) Getxashrate() uint64 {
	return uint64(api.etxash.Hashrate())
}