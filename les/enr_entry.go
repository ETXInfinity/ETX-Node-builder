// Copyright 2019 The go-ETX Authors
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
	"github.com/ETX/go-ETX/core/forkid"
	"github.com/ETX/go-ETX/p2p/dnsdisc"
	"github.com/ETX/go-ETX/p2p/enode"
	"github.com/ETX/go-ETX/rlp"
)

// lesEntry is the "les" ENR entry. This is set for LES servers only.
type lesEntry struct {
	// Ignore additional fields (for forward compatibility).
	VfxVersion uint
	Rest       []rlp.RawValue `rlp:"tail"`
}

func (lesEntry) ENRKey() string { return "les" }

// etxEntry is the "etx" ENR entry. This is redeclared here to avoid depending on package etx.
type etxEntry struct {
	ForkID forkid.ID
	Tail   []rlp.RawValue `rlp:"tail"`
}

func (etxEntry) ENRKey() string { return "etx" }

// setupDiscovery creates the node discovery source for the etx protocol.
func (etx *LightETX) setupDiscovery() (enode.Iterator, error) {
	it := enode.NewFairMix(0)

	// Enable DNS discovery.
	if len(etx.config.etxDiscoveryURLs) != 0 {
		client := dnsdisc.NewClient(dnsdisc.Config{})
		dns, err := client.NewIterator(etx.config.etxDiscoveryURLs...)
		if err != nil {
			return nil, err
		}
		it.AddSource(dns)
	}

	// Enable DHT.
	if etx.udpEnabled {
		it.AddSource(etx.p2pServer.DiscV5.RandomNodes())
	}

	forkFilter := forkid.NewFilter(etx.blockchain)
	iterator := enode.Filter(it, func(n *enode.Node) bool { return nodeIsServer(forkFilter, n) })
	return iterator, nil
}

// nodeIsServer checks whetxer n is an LES server node.
func nodeIsServer(forkFilter forkid.Filter, n *enode.Node) bool {
	var les lesEntry
	var etx etxEntry
	return n.Load(&les) == nil && n.Load(&etx) == nil && forkFilter(etx.ForkID) == nil
}
