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
	"sync/atomic"
	"testing"
	"time"

	"github.com/ETX/go-ETX/etx/downloader"
	"github.com/ETX/go-ETX/etx/protocols/etx"
	"github.com/ETX/go-ETX/etx/protocols/snap"
	"github.com/ETX/go-ETX/p2p"
	"github.com/ETX/go-ETX/p2p/enode"
)

// Tests that snap sync is disabled after a successful sync cycle.
func TestSnapSyncDisabling66(t *testing.T) { testSnapSyncDisabling(t, etx.etx66, snap.SNAP1) }
func TestSnapSyncDisabling67(t *testing.T) { testSnapSyncDisabling(t, etx.etx67, snap.SNAP1) }

// Tests that snap sync gets disabled as soon as a real block is successfully
// imported into the blockchain.
func testSnapSyncDisabling(t *testing.T, etxVer uint, snapVer uint) {
	t.Parallel()

	// Create an empty handler and ensure it's in snap sync mode
	empty := newTestHandler()
	if atomic.LoadUint32(&empty.handler.snapSync) == 0 {
		t.Fatalf("snap sync disabled on pristine blockchain")
	}
	defer empty.close()

	// Create a full handler and ensure snap sync ends up disabled
	full := newTestHandlerWithBlocks(1024)
	if atomic.LoadUint32(&full.handler.snapSync) == 1 {
		t.Fatalf("snap sync not disabled on non-empty blockchain")
	}
	defer full.close()

	// Sync up the two handlers via both `etx` and `snap`
	caps := []p2p.Cap{{Name: "etx", Version: etxVer}, {Name: "snap", Version: snapVer}}

	emptyPipeetx, fullPipeetx := p2p.MsgPipe()
	defer emptyPipeetx.Close()
	defer fullPipeetx.Close()

	emptyPeeretx := etx.NewPeer(etxVer, p2p.NewPeer(enode.ID{1}, "", caps), emptyPipeetx, empty.txpool)
	fullPeeretx := etx.NewPeer(etxVer, p2p.NewPeer(enode.ID{2}, "", caps), fullPipeetx, full.txpool)
	defer emptyPeeretx.Close()
	defer fullPeeretx.Close()

	go empty.handler.runetxPeer(emptyPeeretx, func(peer *etx.Peer) error {
		return etx.Handle((*etxHandler)(empty.handler), peer)
	})
	go full.handler.runetxPeer(fullPeeretx, func(peer *etx.Peer) error {
		return etx.Handle((*etxHandler)(full.handler), peer)
	})

	emptyPipeSnap, fullPipeSnap := p2p.MsgPipe()
	defer emptyPipeSnap.Close()
	defer fullPipeSnap.Close()

	emptyPeerSnap := snap.NewPeer(snapVer, p2p.NewPeer(enode.ID{1}, "", caps), emptyPipeSnap)
	fullPeerSnap := snap.NewPeer(snapVer, p2p.NewPeer(enode.ID{2}, "", caps), fullPipeSnap)

	go empty.handler.runSnapExtension(emptyPeerSnap, func(peer *snap.Peer) error {
		return snap.Handle((*snapHandler)(empty.handler), peer)
	})
	go full.handler.runSnapExtension(fullPeerSnap, func(peer *snap.Peer) error {
		return snap.Handle((*snapHandler)(full.handler), peer)
	})
	// Wait a bit for the above handlers to start
	time.Sleep(250 * time.Millisecond)

	// Check that snap sync was disabled
	op := peerToSyncOp(downloader.SnapSync, empty.handler.peers.peerWithHighestTD())
	if err := empty.handler.doSync(op); err != nil {
		t.Fatal("sync failed:", err)
	}
	if atomic.LoadUint32(&empty.handler.snapSync) == 1 {
		t.Fatalf("snap sync not disabled after successful synchronisation")
	}
}
