// Copyright 2022 The go-ETX Authors
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

package math

import (
	"math/big"
	"testing"

	big2 "github.com/holiman/big"
)

// TestFastModexp tests some cases found during fuzzing.
func TestFastModexp(t *testing.T) {
	for i, tc := range []struct {
		base string
		exp  string
		mod  string
	}{
		{"0xeffffff900002f00", "0x40000000000000", "0x200"},
		{"0xf000", "0x4f900b400080000", "0x400000d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9d9ffffff005aeffd310000000000000000000000000000000000009f9f9f9f0000000000000000000000000800000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000befffa5a5a5fff900002f000040000000000000000000000000000000029d9d9d000000000000009f9f9f00000000000000009f9f9f000000f3a080ab00000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f0000000000002900009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f000000cf000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f000000000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000befffff0000c0800000000800000000000000000000000000000002000000000000009f9f9f0000000000000000008000ff000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000befffa5a5a5fff900002f000040000000000000000000000000000000029d9d9d000000000000009f9f9f00000000000000009f9f9f000000f3a080ab00000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f0000000000002900009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f000000000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f000000000000000000000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000080000000000000000000000000000000000000000000000000000000000000beffffff900002f0000400000c100000000000000000000000000000000000000006160600000000000000000008000ff0000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f00000000000000009f9f0000"},
		{"5", "1435700818", "72"},
		{"0xffff", "0x300030003000300030003000300030003000302a3000300030003000300030003000300030003000300030003000300030003030623066307f3030783062303430383064303630343036", "0x300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
		{"0x3133", "0x667f00000000000000000000000000ff002a000000000000000000000000000000000000000000000000000000000000667fff30783362773057ee756a6c266134643831646230313630", "0x3030000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"},
	} {
		var (
			base, _  = new(big.Int).SetString(tc.base, 0)
			exp, _   = new(big.Int).SetString(tc.exp, 0)
			mod, _   = new(big.Int).SetString(tc.mod, 0)
			base2, _ = new(big2.Int).SetString(tc.base, 0)
			exp2, _  = new(big2.Int).SetString(tc.exp, 0)
			mod2, _  = new(big2.Int).SetString(tc.mod, 0)
		)
		var a = new(big2.Int).Exp(base2, exp2, mod2).String()
		var b = new(big.Int).Exp(base, exp, mod).String()
		if a != b {
			t.Errorf("test %d: %#x ^ %#x mod %#x \n have %x\n want %x", i, base, exp, mod, a, b)
		}
	}
}
