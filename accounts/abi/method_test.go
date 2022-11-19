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

package abi

import (
	"strings"
	"testing"
)

const metxoddata = `
[
	{"type": "function", "name": "balance", "stateMutability": "view"},
	{"type": "function", "name": "send", "inputs": [{ "name": "amount", "type": "uint256" }]},
	{"type": "function", "name": "transfer", "inputs": [{"name": "from", "type": "address"}, {"name": "to", "type": "address"}, {"name": "value", "type": "uint256"}], "outputs": [{"name": "success", "type": "bool"}]},
	{"constant":false,"inputs":[{"components":[{"name":"x","type":"uint256"},{"name":"y","type":"uint256"}],"name":"a","type":"tuple"}],"name":"tuple","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[{"components":[{"name":"x","type":"uint256"},{"name":"y","type":"uint256"}],"name":"a","type":"tuple[]"}],"name":"tupleSlice","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[{"components":[{"name":"x","type":"uint256"},{"name":"y","type":"uint256"}],"name":"a","type":"tuple[5]"}],"name":"tupleArray","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"constant":false,"inputs":[{"components":[{"name":"x","type":"uint256"},{"name":"y","type":"uint256"}],"name":"a","type":"tuple[5][]"}],"name":"complexTuple","outputs":[],"payable":false,"stateMutability":"nonpayable","type":"function"},
	{"stateMutability":"nonpayable","type":"fallback"},
	{"stateMutability":"payable","type":"receive"}
]`

func TestMetxodString(t *testing.T) {
	var table = []struct {
		metxod      string
		expectation string
	}{
		{
			metxod:      "balance",
			expectation: "function balance() view returns()",
		},
		{
			metxod:      "send",
			expectation: "function send(uint256 amount) returns()",
		},
		{
			metxod:      "transfer",
			expectation: "function transfer(address from, address to, uint256 value) returns(bool success)",
		},
		{
			metxod:      "tuple",
			expectation: "function tuple((uint256,uint256) a) returns()",
		},
		{
			metxod:      "tupleArray",
			expectation: "function tupleArray((uint256,uint256)[5] a) returns()",
		},
		{
			metxod:      "tupleSlice",
			expectation: "function tupleSlice((uint256,uint256)[] a) returns()",
		},
		{
			metxod:      "complexTuple",
			expectation: "function complexTuple((uint256,uint256)[5][] a) returns()",
		},
		{
			metxod:      "fallback",
			expectation: "fallback() returns()",
		},
		{
			metxod:      "receive",
			expectation: "receive() payable returns()",
		},
	}

	abi, err := JSON(strings.NewReader(metxoddata))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range table {
		var got string
		if test.metxod == "fallback" {
			got = abi.Fallback.String()
		} else if test.metxod == "receive" {
			got = abi.Receive.String()
		} else {
			got = abi.Metxods[test.metxod].String()
		}
		if got != test.expectation {
			t.Errorf("expected string to be %s, got %s", test.expectation, got)
		}
	}
}

func TestMetxodSig(t *testing.T) {
	var cases = []struct {
		metxod string
		expect string
	}{
		{
			metxod: "balance",
			expect: "balance()",
		},
		{
			metxod: "send",
			expect: "send(uint256)",
		},
		{
			metxod: "transfer",
			expect: "transfer(address,address,uint256)",
		},
		{
			metxod: "tuple",
			expect: "tuple((uint256,uint256))",
		},
		{
			metxod: "tupleArray",
			expect: "tupleArray((uint256,uint256)[5])",
		},
		{
			metxod: "tupleSlice",
			expect: "tupleSlice((uint256,uint256)[])",
		},
		{
			metxod: "complexTuple",
			expect: "complexTuple((uint256,uint256)[5][])",
		},
	}
	abi, err := JSON(strings.NewReader(metxoddata))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range cases {
		got := abi.Metxods[test.metxod].Sig
		if got != test.expect {
			t.Errorf("expected string to be %s, got %s", test.expect, got)
		}
	}
}
