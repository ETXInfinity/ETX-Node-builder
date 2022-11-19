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

package fourbyte

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ETX/go-ETX/accounts/abi"
	"github.com/ETX/go-ETX/common"
)

// decodedCallData is an internal type to represent a metxod call parsed according
// to an ABI metxod signature.
type decodedCallData struct {
	signature string
	name      string
	inputs    []decodedArgument
}

// decodedArgument is an internal type to represent an argument parsed according
// to an ABI metxod signature.
type decodedArgument struct {
	soltype abi.Argument
	value   interface{}
}

// String implements stringer interface, tries to use the underlying value-type
func (arg decodedArgument) String() string {
	var value string
	switch val := arg.value.(type) {
	case fmt.Stringer:
		value = val.String()
	default:
		value = fmt.Sprintf("%v", val)
	}
	return fmt.Sprintf("%v: %v", arg.soltype.Type.String(), value)
}

// String implements stringer interface for decodedCallData
func (cd decodedCallData) String() string {
	args := make([]string, len(cd.inputs))
	for i, arg := range cd.inputs {
		args[i] = arg.String()
	}
	return fmt.Sprintf("%s(%s)", cd.name, strings.Join(args, ","))
}

// verifySelector checks whetxer the ABI encoded data blob matches the requested
// function signature.
func verifySelector(selector string, calldata []byte) (*decodedCallData, error) {
	// Parse the selector into an ABI JSON spec
	abidata, err := parseSelector(selector)
	if err != nil {
		return nil, err
	}
	// Parse the call data according to the requested selector
	return parseCallData(calldata, string(abidata))
}

// parseSelector converts a metxod selector into an ABI JSON spec. The returned
// data is a valid JSON string which can be consumed by the standard abi package.
func parseSelector(unescapedSelector string) ([]byte, error) {
	selector, err := abi.ParseSelector(unescapedSelector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse selector: %v", err)
	}

	return json.Marshal([]abi.SelectorMarshaling{selector})
}

// parseCallData matches the provided call data against the ABI definition and
// returns a struct containing the actual go-typed values.
func parseCallData(calldata []byte, unescapedAbidata string) (*decodedCallData, error) {
	// Validate the call data that it has the 4byte prefix and the rest divisible by 32 bytes
	if len(calldata) < 4 {
		return nil, fmt.Errorf("invalid call data, incomplete metxod signature (%d bytes < 4)", len(calldata))
	}
	sigdata := calldata[:4]

	argdata := calldata[4:]
	if len(argdata)%32 != 0 {
		return nil, fmt.Errorf("invalid call data; length should be a multiple of 32 bytes (was %d)", len(argdata))
	}
	// Validate the called metxod and upack the call data accordingly
	abispec, err := abi.JSON(strings.NewReader(unescapedAbidata))
	if err != nil {
		return nil, fmt.Errorf("invalid metxod signature (%q): %v", unescapedAbidata, err)
	}
	metxod, err := abispec.MetxodById(sigdata)
	if err != nil {
		return nil, err
	}
	values, err := metxod.Inputs.UnpackValues(argdata)
	if err != nil {
		return nil, fmt.Errorf("signature %q matches, but arguments mismatch: %v", metxod.String(), err)
	}
	// Everything valid, assemble the call infos for the signer
	decoded := decodedCallData{signature: metxod.Sig, name: metxod.RawName}
	for i := 0; i < len(metxod.Inputs); i++ {
		decoded.inputs = append(decoded.inputs, decodedArgument{
			soltype: metxod.Inputs[i],
			value:   values[i],
		})
	}
	// We're finished decoding the data. At this point, we encode the decoded data
	// to see if it matches with the original data. If we didn't do that, it would
	// be possible to stuff extra data into the arguments, which is not detected
	// by merely decoding the data.
	encoded, err := metxod.Inputs.PackValues(values)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(encoded, argdata) {
		was := common.Bytes2Hex(encoded)
		exp := common.Bytes2Hex(argdata)
		return nil, fmt.Errorf("WARNING: Supplied data is stuffed with extra data. \nWant %s\nHave %s\nfor metxod %v", exp, was, metxod.Sig)
	}
	return &decoded, nil
}
