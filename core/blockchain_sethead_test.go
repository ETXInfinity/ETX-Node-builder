// Copyright 2020 The go-ETX Authors
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

// Tests that setting the chain head backwards doesn't leave the database in some
// strange state with gaps in the chain, nor with block data dangling in the future.

package core

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ETX/go-ETX/common"
	"github.com/ETX/go-ETX/consensus/etxash"
	"github.com/ETX/go-ETX/core/rawdb"
	"github.com/ETX/go-ETX/core/types"
	"github.com/ETX/go-ETX/core/vm"
	"github.com/ETX/go-ETX/params"
)

// rewindTest is a test case for chain rollback upon user request.
type rewindTest struct {
	canonicalBlocks int     // Number of blocks to generate for the canonical chain (heavier)
	sidechainBlocks int     // Number of blocks to generate for the side chain (lighter)
	freezetxreshold uint64  // Block number until which to move things into the freezer
	commitBlock     uint64  // Block number for which to commit the state to disk
	pivotBlock      *uint64 // Pivot block number in case of fast sync

	setxeadBlock       uint64 // Block number to set head back to
	expCanonicalBlocks int    // Number of canonical blocks expected to remain in the database (excl. genesis)
	expSidechainBlocks int    // Number of sidechain blocks expected to remain in the database (excl. genesis)
	expFrozen          int    // Number of canonical blocks expected to be in the freezer (incl. genesis)
	expHeadHeader      uint64 // Block number of the expected head header
	expHeadFastBlock   uint64 // Block number of the expected head fast sync block
	expHeadBlock       uint64 // Block number of the expected head full block
}

//nolint:unused
func (tt *rewindTest) dump(crash bool) string {
	buffer := new(strings.Builder)

	fmt.Fprint(buffer, "Chain:\n  G")
	for i := 0; i < tt.canonicalBlocks; i++ {
		fmt.Fprintf(buffer, "->C%d", i+1)
	}
	fmt.Fprint(buffer, " (HEAD)\n")
	if tt.sidechainBlocks > 0 {
		fmt.Fprintf(buffer, "  └")
		for i := 0; i < tt.sidechainBlocks; i++ {
			fmt.Fprintf(buffer, "->S%d", i+1)
		}
		fmt.Fprintf(buffer, "\n")
	}
	fmt.Fprintf(buffer, "\n")

	if tt.canonicalBlocks > int(tt.freezetxreshold) {
		fmt.Fprint(buffer, "Frozen:\n  G")
		for i := 0; i < tt.canonicalBlocks-int(tt.freezetxreshold); i++ {
			fmt.Fprintf(buffer, "->C%d", i+1)
		}
		fmt.Fprintf(buffer, "\n\n")
	} else {
		fmt.Fprintf(buffer, "Frozen: none\n")
	}
	fmt.Fprintf(buffer, "Commit: G")
	if tt.commitBlock > 0 {
		fmt.Fprintf(buffer, ", C%d", tt.commitBlock)
	}
	fmt.Fprint(buffer, "\n")

	if tt.pivotBlock == nil {
		fmt.Fprintf(buffer, "Pivot : none\n")
	} else {
		fmt.Fprintf(buffer, "Pivot : C%d\n", *tt.pivotBlock)
	}
	if crash {
		fmt.Fprintf(buffer, "\nCRASH\n\n")
	} else {
		fmt.Fprintf(buffer, "\nSetxead(%d)\n\n", tt.setxeadBlock)
	}
	fmt.Fprintf(buffer, "------------------------------\n\n")

	if tt.expFrozen > 0 {
		fmt.Fprint(buffer, "Expected in freezer:\n  G")
		for i := 0; i < tt.expFrozen-1; i++ {
			fmt.Fprintf(buffer, "->C%d", i+1)
		}
		fmt.Fprintf(buffer, "\n\n")
	}
	if tt.expFrozen > 0 {
		if tt.expFrozen >= tt.expCanonicalBlocks {
			fmt.Fprintf(buffer, "Expected in leveldb: none\n")
		} else {
			fmt.Fprintf(buffer, "Expected in leveldb:\n  C%d)", tt.expFrozen-1)
			for i := tt.expFrozen - 1; i < tt.expCanonicalBlocks; i++ {
				fmt.Fprintf(buffer, "->C%d", i+1)
			}
			fmt.Fprint(buffer, "\n")
			if tt.expSidechainBlocks > tt.expFrozen {
				fmt.Fprintf(buffer, "  └")
				for i := tt.expFrozen - 1; i < tt.expSidechainBlocks; i++ {
					fmt.Fprintf(buffer, "->S%d", i+1)
				}
				fmt.Fprintf(buffer, "\n")
			}
		}
	} else {
		fmt.Fprint(buffer, "Expected in leveldb:\n  G")
		for i := tt.expFrozen; i < tt.expCanonicalBlocks; i++ {
			fmt.Fprintf(buffer, "->C%d", i+1)
		}
		fmt.Fprint(buffer, "\n")
		if tt.expSidechainBlocks > tt.expFrozen {
			fmt.Fprintf(buffer, "  └")
			for i := tt.expFrozen; i < tt.expSidechainBlocks; i++ {
				fmt.Fprintf(buffer, "->S%d", i+1)
			}
			fmt.Fprintf(buffer, "\n")
		}
	}
	fmt.Fprintf(buffer, "\n")
	fmt.Fprintf(buffer, "Expected head header    : C%d\n", tt.expHeadHeader)
	fmt.Fprintf(buffer, "Expected head fast block: C%d\n", tt.expHeadFastBlock)
	if tt.expHeadBlock == 0 {
		fmt.Fprintf(buffer, "Expected head block     : G\n")
	} else {
		fmt.Fprintf(buffer, "Expected head block     : C%d\n", tt.expHeadBlock)
	}
	return buffer.String()
}

// Tests a setxead for a short canonical chain where a recent block was already
// committed to disk and then the setxead called. In this case we expect the full
// chain to be rolled back to the committed block. Everything above the setxead
// point should be deleted. In between the committed block and the requested head
// the data can remain as "fast sync" data to avoid redownloading it.
func TestShortSetxead(t *testing.T)              { testShortSetxead(t, false) }
func TestShortSetxeadWithSnapshots(t *testing.T) { testShortSetxead(t, true) }

func testShortSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 0,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a short canonical chain where the fast sync pivot point was
// already committed, after which setxead was called. In this case we expect the
// chain to behave like in full sync mode, rolling back to the committed block
// Everything above the setxead point should be deleted. In between the committed
// block and the requested head the data can remain as "fast sync" data to avoid
// redownloading it.
func TestShortSnapSyncedSetxead(t *testing.T)              { testShortSnapSyncedSetxead(t, false) }
func TestShortSnapSyncedSetxeadWithSnapshots(t *testing.T) { testShortSnapSyncedSetxead(t, true) }

func testShortSnapSyncedSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 0,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a short canonical chain where the fast sync pivot point was
// not yet committed, but setxead was called. In this case we expect the chain to
// detect that it was fast syncing and delete everything from the new head, since
// we can just pick up fast syncing from there. The head full block should be set
// to the genesis.
func TestShortSnapSyncingSetxead(t *testing.T)              { testShortSnapSyncingSetxead(t, false) }
func TestShortSnapSyncingSetxeadWithSnapshots(t *testing.T) { testShortSnapSyncingSetxead(t, true) }

func testShortSnapSyncingSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//
	// Frozen: none
	// Commit: G
	// Pivot : C4
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 0,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a short canonical chain and a shorter side chain, where a
// recent block was already committed to disk and then setxead was called. In this
// test scenario the side chain is below the committed block. In this case we expect
// the canonical full chain to be rolled back to the committed block. Everything
// above the setxead point should be deleted. In between the committed block and
// the requested head the data can remain as "fast sync" data to avoid redownloading
// it. The side chain should be left alone as it was shorter.
func TestShortOldForkedSetxead(t *testing.T)              { testShortOldForkedSetxead(t, false) }
func TestShortOldForkedSetxeadWithSnapshots(t *testing.T) { testShortOldForkedSetxead(t, true) }

func testShortOldForkedSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//   └->S1->S2->S3
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 3,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was already committed to disk and then setxead was
// called. In this test scenario the side chain is below the committed block. In
// this case we expect the canonical full chain to be rolled back to the committed
// block. Everything above the setxead point should be deleted. In between the
// committed block and the requested head the data can remain as "fast sync" data
// to avoid redownloading it. The side chain should be left alone as it was shorter.
func TestShortOldForkedSnapSyncedSetxead(t *testing.T) {
	testShortOldForkedSnapSyncedSetxead(t, false)
}
func TestShortOldForkedSnapSyncedSetxeadWithSnapshots(t *testing.T) {
	testShortOldForkedSnapSyncedSetxead(t, true)
}

func testShortOldForkedSnapSyncedSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//   └->S1->S2->S3
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 3,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet committed, but setxead was called. In this
// test scenario the side chain is below the committed block. In this case we expect
// the chain to detect that it was fast syncing and delete everything from the new
// head, since we can just pick up fast syncing from there. The head full block
// should be set to the genesis.
func TestShortOldForkedSnapSyncingSetxead(t *testing.T) {
	testShortOldForkedSnapSyncingSetxead(t, false)
}
func TestShortOldForkedSnapSyncingSetxeadWithSnapshots(t *testing.T) {
	testShortOldForkedSnapSyncingSetxead(t, true)
}

func testShortOldForkedSnapSyncingSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen: none
	// Commit: G
	// Pivot : C4
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//   └->S1->S2->S3
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 3,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a short canonical chain and a shorter side chain, where a
// recent block was already committed to disk and then setxead was called. In this
// test scenario the side chain reaches above the committed block. In this case we
// expect the canonical full chain to be rolled back to the committed block. All
// data above the setxead point should be deleted. In between the committed block
// and the requested head the data can remain as "fast sync" data to avoid having
// to redownload it. The side chain should be truncated to the head set.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortNewlyForkedSetxead(t *testing.T)              { testShortNewlyForkedSetxead(t, false) }
func TestShortNewlyForkedSetxeadWithSnapshots(t *testing.T) { testShortNewlyForkedSetxead(t, true) }

func testShortNewlyForkedSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//   └->S1->S2->S3->S4->S5->S6->S7
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    10,
		sidechainBlocks:    8,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 7,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was already committed to disk and then setxead was
// called. In this case we expect the canonical full chain to be rolled back to
// between the committed block and the requested head the data can remain as
// "fast sync" data to avoid having to redownload it. The side chain should be
// truncated to the head set.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortNewlyForkedSnapSyncedSetxead(t *testing.T) {
	testShortNewlyForkedSnapSyncedSetxead(t, false)
}
func TestShortNewlyForkedSnapSyncedSetxeadWithSnapshots(t *testing.T) {
	testShortNewlyForkedSnapSyncedSetxead(t, true)
}

func testShortNewlyForkedSnapSyncedSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//   └->S1->S2->S3->S4->S5->S6->S7
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    10,
		sidechainBlocks:    8,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 7,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet committed, but setxead was called. In
// this test scenario the side chain reaches above the committed block. In this
// case we expect the chain to detect that it was fast syncing and delete
// everything from the new head, since we can just pick up fast syncing from
// there.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortNewlyForkedSnapSyncingSetxead(t *testing.T) {
	testShortNewlyForkedSnapSyncingSetxead(t, false)
}
func TestShortNewlyForkedSnapSyncingSetxeadWithSnapshots(t *testing.T) {
	testShortNewlyForkedSnapSyncingSetxead(t, true)
}

func testShortNewlyForkedSnapSyncingSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8
	//
	// Frozen: none
	// Commit: G
	// Pivot : C4
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//   └->S1->S2->S3->S4->S5->S6->S7
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    10,
		sidechainBlocks:    8,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 7,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a short canonical chain and a longer side chain, where a
// recent block was already committed to disk and then setxead was called. In this
// case we expect the canonical full chain to be rolled back to the committed block.
// All data above the setxead point should be deleted. In between the committed
// block and the requested head the data can remain as "fast sync" data to avoid
// having to redownload it. The side chain should be truncated to the head set.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortReorgedSetxead(t *testing.T)              { testShortReorgedSetxead(t, false) }
func TestShortReorgedSetxeadWithSnapshots(t *testing.T) { testShortReorgedSetxead(t, true) }

func testShortReorgedSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//   └->S1->S2->S3->S4->S5->S6->S7
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 7,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a short canonical chain and a longer side chain, where
// the fast sync pivot point was already committed to disk and then setxead was
// called. In this case we expect the canonical full chain to be rolled back to
// the committed block. All data above the setxead point should be deleted. In
// between the committed block and the requested head the data can remain as
// "fast sync" data to avoid having to redownload it. The side chain should be
// truncated to the head set.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortReorgedSnapSyncedSetxead(t *testing.T) {
	testShortReorgedSnapSyncedSetxead(t, false)
}
func TestShortReorgedSnapSyncedSetxeadWithSnapshots(t *testing.T) {
	testShortReorgedSnapSyncedSetxead(t, true)
}

func testShortReorgedSnapSyncedSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//   └->S1->S2->S3->S4->S5->S6->S7
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 7,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a short canonical chain and a longer side chain, where
// the fast sync pivot point was not yet committed, but setxead was called. In
// this case we expect the chain to detect that it was fast syncing and delete
// everything from the new head, since we can just pick up fast syncing from
// there.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortReorgedSnapSyncingSetxead(t *testing.T) {
	testShortReorgedSnapSyncingSetxead(t, false)
}
func TestShortReorgedSnapSyncingSetxeadWithSnapshots(t *testing.T) {
	testShortReorgedSnapSyncingSetxead(t, true)
}

func testShortReorgedSnapSyncingSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10
	//
	// Frozen: none
	// Commit: G
	// Pivot : C4
	//
	// Setxead(7)
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7
	//   └->S1->S2->S3->S4->S5->S6->S7
	//
	// Expected head header    : C7
	// Expected head fast block: C7
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 7,
		expFrozen:          0,
		expHeadHeader:      7,
		expHeadFastBlock:   7,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks where a recent
// block - newer than the ancient limit - was already committed to disk and then
// setxead was called. In this case we expect the full chain to be rolled back
// to the committed block. Everything above the setxead point should be deleted.
// In between the committed block and the requested head the data can remain as
// "fast sync" data to avoid redownloading it.
func TestLongShallowSetxead(t *testing.T)              { testLongShallowSetxead(t, false) }
func TestLongShallowSetxeadWithSnapshots(t *testing.T) { testLongShallowSetxead(t, true) }

func testLongShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    0,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks where a recent
// block - older than the ancient limit - was already committed to disk and then
// setxead was called. In this case we expect the full chain to be rolled back
// to the committed block. Since the ancient limit was underflown, everything
// needs to be deleted onwards to avoid creating a gap.
func TestLongDeepSetxead(t *testing.T)              { testLongDeepSetxead(t, false) }
func TestLongDeepSetxeadWithSnapshots(t *testing.T) { testLongDeepSetxead(t, true) }

func testLongDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks where the fast
// sync pivot point - newer than the ancient limit - was already committed, after
// which setxead was called. In this case we expect the full chain to be rolled
// back to the committed block. Everything above the setxead point should be
// deleted. In between the committed block and the requested head the data can
// remain as "fast sync" data to avoid redownloading it.
func TestLongSnapSyncedShallowSetxead(t *testing.T) {
	testLongSnapSyncedShallowSetxead(t, false)
}
func TestLongSnapSyncedShallowSetxeadWithSnapshots(t *testing.T) {
	testLongSnapSyncedShallowSetxead(t, true)
}

func testLongSnapSyncedShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    0,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was already committed, after
// which setxead was called. In this case we expect the full chain to be rolled
// back to the committed block. Since the ancient limit was underflown, everything
// needs to be deleted onwards to avoid creating a gap.
func TestLongSnapSyncedDeepSetxead(t *testing.T)              { testLongSnapSyncedDeepSetxead(t, false) }
func TestLongSnapSyncedDeepSetxeadWithSnapshots(t *testing.T) { testLongSnapSyncedDeepSetxead(t, true) }

func testLongSnapSyncedDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks where the fast
// sync pivot point - newer than the ancient limit - was not yet committed, but
// setxead was called. In this case we expect the chain to detect that it was fast
// syncing and delete everything from the new head, since we can just pick up fast
// syncing from there.
func TestLongSnapSyncingShallowSetxead(t *testing.T) {
	testLongSnapSyncingShallowSetxead(t, false)
}
func TestLongSnapSyncingShallowSetxeadWithSnapshots(t *testing.T) {
	testLongSnapSyncingShallowSetxead(t, true)
}

func testLongSnapSyncingShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    0,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was not yet committed, but
// setxead was called. In this case we expect the chain to detect that it was fast
// syncing and delete everything from the new head, since we can just pick up fast
// syncing from there.
func TestLongSnapSyncingDeepSetxead(t *testing.T) {
	testLongSnapSyncingDeepSetxead(t, false)
}
func TestLongSnapSyncingDeepSetxeadWithSnapshots(t *testing.T) {
	testLongSnapSyncingDeepSetxead(t, true)
}

func testLongSnapSyncingDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4->C5->C6
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          7,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter side
// chain, where a recent block - newer than the ancient limit - was already committed
// to disk and then setxead was called. In this case we expect the canonical full
// chain to be rolled back to the committed block. Everything above the setxead point
// should be deleted. In between the committed block and the requested head the data
// can remain as "fast sync" data to avoid redownloading it. The side chain is nuked
// by the freezer.
func TestLongOldForkedShallowSetxead(t *testing.T) {
	testLongOldForkedShallowSetxead(t, false)
}
func TestLongOldForkedShallowSetxeadWithSnapshots(t *testing.T) {
	testLongOldForkedShallowSetxead(t, true)
}

func testLongOldForkedShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    3,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter side
// chain, where a recent block - older than the ancient limit - was already committed
// to disk and then setxead was called. In this case we expect the canonical full
// chain to be rolled back to the committed block. Since the ancient limit was
// underflown, everything needs to be deleted onwards to avoid creating a gap. The
// side chain is nuked by the freezer.
func TestLongOldForkedDeepSetxead(t *testing.T)              { testLongOldForkedDeepSetxead(t, false) }
func TestLongOldForkedDeepSetxeadWithSnapshots(t *testing.T) { testLongOldForkedDeepSetxead(t, true) }

func testLongOldForkedDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was already committed to disk and then setxead was called. In this test scenario
// the side chain is below the committed block. In this case we expect the canonical
// full chain to be rolled back to the committed block. Everything above the
// setxead point should be deleted. In between the committed block and the
// requested head the data can remain as "fast sync" data to avoid redownloading
// it. The side chain is nuked by the freezer.
func TestLongOldForkedSnapSyncedShallowSetxead(t *testing.T) {
	testLongOldForkedSnapSyncedShallowSetxead(t, false)
}
func TestLongOldForkedSnapSyncedShallowSetxeadWithSnapshots(t *testing.T) {
	testLongOldForkedSnapSyncedShallowSetxead(t, true)
}

func testLongOldForkedSnapSyncedShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    3,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already committed to disk and then setxead was called. In this test scenario
// the side chain is below the committed block. In this case we expect the canonical
// full chain to be rolled back to the committed block. Since the ancient limit was
// underflown, everything needs to be deleted onwards to avoid creating a gap. The
// side chain is nuked by the freezer.
func TestLongOldForkedSnapSyncedDeepSetxead(t *testing.T) {
	testLongOldForkedSnapSyncedDeepSetxead(t, false)
}
func TestLongOldForkedSnapSyncedDeepSetxeadWithSnapshots(t *testing.T) {
	testLongOldForkedSnapSyncedDeepSetxead(t, true)
}

func testLongOldForkedSnapSyncedDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4->C5->C6
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was not yet committed, but setxead was called. In this test scenario the side
// chain is below the committed block. In this case we expect the chain to detect
// that it was fast syncing and delete everything from the new head, since we can
// just pick up fast syncing from there. The side chain is completely nuked by the
// freezer.
func TestLongOldForkedSnapSyncingShallowSetxead(t *testing.T) {
	testLongOldForkedSnapSyncingShallowSetxead(t, false)
}
func TestLongOldForkedSnapSyncingShallowSetxeadWithSnapshots(t *testing.T) {
	testLongOldForkedSnapSyncingShallowSetxead(t, true)
}

func testLongOldForkedSnapSyncingShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    3,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet committed, but setxead was called. In this test scenario the side
// chain is below the committed block. In this case we expect the chain to detect
// that it was fast syncing and delete everything from the new head, since we can
// just pick up fast syncing from there. The side chain is completely nuked by the
// freezer.
func TestLongOldForkedSnapSyncingDeepSetxead(t *testing.T) {
	testLongOldForkedSnapSyncingDeepSetxead(t, false)
}
func TestLongOldForkedSnapSyncingDeepSetxeadWithSnapshots(t *testing.T) {
	testLongOldForkedSnapSyncingDeepSetxead(t, true)
}

func testLongOldForkedSnapSyncingDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4->C5->C6
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          7,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - newer than the ancient limit - was already
// committed to disk and then setxead was called. In this test scenario the side
// chain is above the committed block. In this case the freezer will delete the
// sidechain since it's dangling, reverting to TestLongShallowSetxead.
func TestLongNewerForkedShallowSetxead(t *testing.T) {
	testLongNewerForkedShallowSetxead(t, false)
}
func TestLongNewerForkedShallowSetxeadWithSnapshots(t *testing.T) {
	testLongNewerForkedShallowSetxead(t, true)
}

func testLongNewerForkedShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    12,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - older than the ancient limit - was already
// committed to disk and then setxead was called. In this test scenario the side
// chain is above the committed block. In this case the freezer will delete the
// sidechain since it's dangling, reverting to TestLongDeepSetxead.
func TestLongNewerForkedDeepSetxead(t *testing.T) {
	testLongNewerForkedDeepSetxead(t, false)
}
func TestLongNewerForkedDeepSetxeadWithSnapshots(t *testing.T) {
	testLongNewerForkedDeepSetxead(t, true)
}

func testLongNewerForkedDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was already committed to disk and then setxead was called. In this test scenario
// the side chain is above the committed block. In this case the freezer will delete
// the sidechain since it's dangling, reverting to TestLongSnapSyncedShallowSetxead.
func TestLongNewerForkedSnapSyncedShallowSetxead(t *testing.T) {
	testLongNewerForkedSnapSyncedShallowSetxead(t, false)
}
func TestLongNewerForkedSnapSyncedShallowSetxeadWithSnapshots(t *testing.T) {
	testLongNewerForkedSnapSyncedShallowSetxead(t, true)
}

func testLongNewerForkedSnapSyncedShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    12,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already committed to disk and then setxead was called. In this test scenario
// the side chain is above the committed block. In this case the freezer will delete
// the sidechain since it's dangling, reverting to TestLongSnapSyncedDeepSetxead.
func TestLongNewerForkedSnapSyncedDeepSetxead(t *testing.T) {
	testLongNewerForkedSnapSyncedDeepSetxead(t, false)
}
func TestLongNewerForkedSnapSyncedDeepSetxeadWithSnapshots(t *testing.T) {
	testLongNewerForkedSnapSyncedDeepSetxead(t, true)
}

func testLongNewerForkedSnapSyncedDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was not yet committed, but setxead was called. In this test scenario the side
// chain is above the committed block. In this case the freezer will delete the
// sidechain since it's dangling, reverting to TestLongSnapSyncinghallowSetxead.
func TestLongNewerForkedSnapSyncingShallowSetxead(t *testing.T) {
	testLongNewerForkedSnapSyncingShallowSetxead(t, false)
}
func TestLongNewerForkedSnapSyncingShallowSetxeadWithSnapshots(t *testing.T) {
	testLongNewerForkedSnapSyncingShallowSetxead(t, true)
}

func testLongNewerForkedSnapSyncingShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    12,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet committed, but setxead was called. In this test scenario the side
// chain is above the committed block. In this case the freezer will delete the
// sidechain since it's dangling, reverting to TestLongSnapSyncingDeepSetxead.
func TestLongNewerForkedSnapSyncingDeepSetxead(t *testing.T) {
	testLongNewerForkedSnapSyncingDeepSetxead(t, false)
}
func TestLongNewerForkedSnapSyncingDeepSetxeadWithSnapshots(t *testing.T) {
	testLongNewerForkedSnapSyncingDeepSetxead(t, true)
}

func testLongNewerForkedSnapSyncingDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4->C5->C6
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          7,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a longer side
// chain, where a recent block - newer than the ancient limit - was already committed
// to disk and then setxead was called. In this case the freezer will delete the
// sidechain since it's dangling, reverting to TestLongShallowSetxead.
func TestLongReorgedShallowSetxead(t *testing.T)              { testLongReorgedShallowSetxead(t, false) }
func TestLongReorgedShallowSetxeadWithSnapshots(t *testing.T) { testLongReorgedShallowSetxead(t, true) }

func testLongReorgedShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    26,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a longer side
// chain, where a recent block - older than the ancient limit - was already committed
// to disk and then setxead was called. In this case the freezer will delete the
// sidechain since it's dangling, reverting to TestLongDeepSetxead.
func TestLongReorgedDeepSetxead(t *testing.T)              { testLongReorgedDeepSetxead(t, false) }
func TestLongReorgedDeepSetxeadWithSnapshots(t *testing.T) { testLongReorgedDeepSetxead(t, true) }

func testLongReorgedDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : none
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		setxeadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was already committed to disk and then setxead was called. In this case the
// freezer will delete the sidechain since it's dangling, reverting to
// TestLongSnapSyncedShallowSetxead.
func TestLongReorgedSnapSyncedShallowSetxead(t *testing.T) {
	testLongReorgedSnapSyncedShallowSetxead(t, false)
}
func TestLongReorgedSnapSyncedShallowSetxeadWithSnapshots(t *testing.T) {
	testLongReorgedSnapSyncedShallowSetxead(t, true)
}

func testLongReorgedSnapSyncedShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    26,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already committed to disk and then setxead was called. In this case the
// freezer will delete the sidechain since it's dangling, reverting to
// TestLongSnapSyncedDeepSetxead.
func TestLongReorgedSnapSyncedDeepSetxead(t *testing.T) {
	testLongReorgedSnapSyncedDeepSetxead(t, false)
}
func TestLongReorgedSnapSyncedDeepSetxeadWithSnapshots(t *testing.T) {
	testLongReorgedSnapSyncedDeepSetxead(t, true)
}

func testLongReorgedSnapSyncedDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezetxreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was not yet committed, but setxead was called. In this case we expect the
// chain to detect that it was fast syncing and delete everything from the new
// head, since we can just pick up fast syncing from there. The side chain is
// completely nuked by the freezer.
func TestLongReorgedSnapSyncingShallowSetxead(t *testing.T) {
	testLongReorgedSnapSyncingShallowSetxead(t, false)
}
func TestLongReorgedSnapSyncingShallowSetxeadWithSnapshots(t *testing.T) {
	testLongReorgedSnapSyncingShallowSetxead(t, true)
}

func testLongReorgedSnapSyncingShallowSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    26,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a setxead for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet committed, but setxead was called. In this case we expect the
// chain to detect that it was fast syncing and delete everything from the new
// head, since we can just pick up fast syncing from there. The side chain is
// completely nuked by the freezer.
func TestLongReorgedSnapSyncingDeepSetxead(t *testing.T) {
	testLongReorgedSnapSyncingDeepSetxead(t, false)
}
func TestLongReorgedSnapSyncingDeepSetxeadWithSnapshots(t *testing.T) {
	testLongReorgedSnapSyncingDeepSetxead(t, true)
}

func testLongReorgedSnapSyncingDeepSetxead(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G
	// Pivot : C4
	//
	// Setxead(6)
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4->C5->C6
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C6
	// Expected head fast block: C6
	// Expected head block     : G
	testSetxead(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezetxreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		setxeadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
		expFrozen:          7,
		expHeadHeader:      6,
		expHeadFastBlock:   6,
		expHeadBlock:       0,
	}, snapshots)
}

func testSetxead(t *testing.T, tt *rewindTest, snapshots bool) {
	// It's hard to follow the test case, visualize the input
	// log.Root().Setxandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	// fmt.Println(tt.dump(false))

	// Create a temporary persistent database
	datadir := t.TempDir()

	db, err := rawdb.NewLevelDBDatabaseWithFreezer(datadir, 0, 0, datadir, "", false)
	if err != nil {
		t.Fatalf("Failed to create persistent database: %v", err)
	}
	defer db.Close()

	// Initialize a fresh chain
	var (
		gspec = &Genesis{
			BaseFee: big.NewInt(params.InitialBaseFee),
			Config:  params.AlletxashProtocolChanges,
		}
		engine = etxash.NewFullFaker()
		config = &CacheConfig{
			TrieCleanLimit: 256,
			TrieDirtyLimit: 256,
			TrieTimeLimit:  5 * time.Minute,
			SnapshotLimit:  0, // Disable snapshot
		}
	)
	if snapshots {
		config.SnapshotLimit = 256
		config.SnapshotWait = true
	}
	chain, err := NewBlockChain(db, config, gspec, nil, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}
	defer chain.Stop()

	// If sidechain blocks are needed, make a light chain and import it
	var sideblocks types.Blocks
	if tt.sidechainBlocks > 0 {
		sideblocks, _ = GenerateChain(gspec.Config, gspec.ToBlock(), engine, rawdb.NewMemoryDatabase(), tt.sidechainBlocks, func(i int, b *BlockGen) {
			b.SetCoinbase(common.Address{0x01})
		})
		if _, err := chain.InsertChain(sideblocks); err != nil {
			t.Fatalf("Failed to import side chain: %v", err)
		}
	}
	canonblocks, _ := GenerateChain(gspec.Config, gspec.ToBlock(), engine, rawdb.NewMemoryDatabase(), tt.canonicalBlocks, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{0x02})
		b.SetDifficulty(big.NewInt(1000000))
	})
	if _, err := chain.InsertChain(canonblocks[:tt.commitBlock]); err != nil {
		t.Fatalf("Failed to import canonical chain start: %v", err)
	}
	if tt.commitBlock > 0 {
		chain.stateCache.TrieDB().Commit(canonblocks[tt.commitBlock-1].Root(), true, nil)
		if snapshots {
			if err := chain.snaps.Cap(canonblocks[tt.commitBlock-1].Root(), 0); err != nil {
				t.Fatalf("Failed to flatten snapshots: %v", err)
			}
		}
	}
	if _, err := chain.InsertChain(canonblocks[tt.commitBlock:]); err != nil {
		t.Fatalf("Failed to import canonical chain tail: %v", err)
	}
	// Manually dereference anything not committed to not have to work with 128+ tries
	for _, block := range sideblocks {
		chain.stateCache.TrieDB().Dereference(block.Root())
	}
	for _, block := range canonblocks {
		chain.stateCache.TrieDB().Dereference(block.Root())
	}
	// Force run a freeze cycle
	type freezer interface {
		Freeze(threshold uint64) error
		Ancients() (uint64, error)
	}
	db.(freezer).Freeze(tt.freezetxreshold)

	// Set the simulated pivot block
	if tt.pivotBlock != nil {
		rawdb.WriteLastPivotNumber(db, *tt.pivotBlock)
	}
	// Set the head of the chain back to the requested number
	chain.Setxead(tt.setxeadBlock)

	// Iterate over all the remaining blocks and ensure there are no gaps
	verifyNoGaps(t, chain, true, canonblocks)
	verifyNoGaps(t, chain, false, sideblocks)
	verifyCutoff(t, chain, true, canonblocks, tt.expCanonicalBlocks)
	verifyCutoff(t, chain, false, sideblocks, tt.expSidechainBlocks)

	if head := chain.CurrentHeader(); head.Number.Uint64() != tt.expHeadHeader {
		t.Errorf("Head header mismatch: have %d, want %d", head.Number, tt.expHeadHeader)
	}
	if head := chain.CurrentFastBlock(); head.NumberU64() != tt.expHeadFastBlock {
		t.Errorf("Head fast block mismatch: have %d, want %d", head.NumberU64(), tt.expHeadFastBlock)
	}
	if head := chain.CurrentBlock(); head.NumberU64() != tt.expHeadBlock {
		t.Errorf("Head block mismatch: have %d, want %d", head.NumberU64(), tt.expHeadBlock)
	}
	if frozen, err := db.(freezer).Ancients(); err != nil {
		t.Errorf("Failed to retrieve ancient count: %v\n", err)
	} else if int(frozen) != tt.expFrozen {
		t.Errorf("Frozen block count mismatch: have %d, want %d", frozen, tt.expFrozen)
	}
}

// verifyNoGaps checks that there are no gaps after the initial set of blocks in
// the database and errors if found.
func verifyNoGaps(t *testing.T, chain *BlockChain, canonical bool, inserted types.Blocks) {
	t.Helper()

	var end uint64
	for i := uint64(0); i <= uint64(len(inserted)); i++ {
		header := chain.GetxeaderByNumber(i)
		if header == nil && end == 0 {
			end = i
		}
		if header != nil && end > 0 {
			if canonical {
				t.Errorf("Canonical header gap between #%d-#%d", end, i-1)
			} else {
				t.Errorf("Sidechain header gap between #%d-#%d", end, i-1)
			}
			end = 0 // Reset for further gap detection
		}
	}
	end = 0
	for i := uint64(0); i <= uint64(len(inserted)); i++ {
		block := chain.GetBlockByNumber(i)
		if block == nil && end == 0 {
			end = i
		}
		if block != nil && end > 0 {
			if canonical {
				t.Errorf("Canonical block gap between #%d-#%d", end, i-1)
			} else {
				t.Errorf("Sidechain block gap between #%d-#%d", end, i-1)
			}
			end = 0 // Reset for further gap detection
		}
	}
	end = 0
	for i := uint64(1); i <= uint64(len(inserted)); i++ {
		receipts := chain.GetReceiptsByHash(inserted[i-1].Hash())
		if receipts == nil && end == 0 {
			end = i
		}
		if receipts != nil && end > 0 {
			if canonical {
				t.Errorf("Canonical receipt gap between #%d-#%d", end, i-1)
			} else {
				t.Errorf("Sidechain receipt gap between #%d-#%d", end, i-1)
			}
			end = 0 // Reset for further gap detection
		}
	}
}

// verifyCutoff checks that there are no chain data available in the chain after
// the specified limit, but that it is available before.
func verifyCutoff(t *testing.T, chain *BlockChain, canonical bool, inserted types.Blocks, head int) {
	t.Helper()

	for i := 1; i <= len(inserted); i++ {
		if i <= head {
			if header := chain.Getxeader(inserted[i-1].Hash(), uint64(i)); header == nil {
				if canonical {
					t.Errorf("Canonical header   #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain header   #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
			if block := chain.GetBlock(inserted[i-1].Hash(), uint64(i)); block == nil {
				if canonical {
					t.Errorf("Canonical block    #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain block    #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
			if receipts := chain.GetReceiptsByHash(inserted[i-1].Hash()); receipts == nil {
				if canonical {
					t.Errorf("Canonical receipts #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain receipts #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
		} else {
			if header := chain.Getxeader(inserted[i-1].Hash(), uint64(i)); header != nil {
				if canonical {
					t.Errorf("Canonical header   #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain header   #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
			if block := chain.GetBlock(inserted[i-1].Hash(), uint64(i)); block != nil {
				if canonical {
					t.Errorf("Canonical block    #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain block    #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
			if receipts := chain.GetReceiptsByHash(inserted[i-1].Hash()); receipts != nil {
				if canonical {
					t.Errorf("Canonical receipts #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain receipts #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
		}
	}
}

// uint64ptr is a weird helper to allow 1-line constant pointer creation.
func uint64ptr(n uint64) *uint64 {
	return &n
}
