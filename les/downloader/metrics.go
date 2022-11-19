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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/ETX/go-ETX/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("etx/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("etx/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("etx/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("etx/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("etx/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("etx/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("etx/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("etx/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("etx/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("etx/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("etx/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("etx/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("etx/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("etx/downloader/states/drop", nil)

	throttleCounter = metrics.NewRegisteredCounter("etx/downloader/throttle", nil)
)
