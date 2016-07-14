// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb20x

import (
	"fmt"
	"testing"

	c "github.com/smartystreets/goconvey/convey"
)

func TestConvert10VRangeBinaryValueToVoltage(t *testing.T) {
	testCases := []struct {
		voltage      float32
		voltageRange VoltageRange
		binaryValue  []byte
	}{
		{-10.0, Range10V, []byte{0, 0}},
		{0.0, Range10V, []byte{0x00, 0x80}},
		{9.999694824, Range10V, []byte{0xff, 0xff}},
	}
	c.Convey("Given the need to convert binary values into voltages", t, func() {
		for _, testCase := range testCases {
			conveyance := fmt.Sprintf("When the binary value is %#x (little endian)", testCase.binaryValue)
			c.Convey(conveyance, func() {
				conveyance := fmt.Sprintf("Then the voltage should be %.4f V", testCase.voltage)
				c.Convey(conveyance, func() {
					computedValue, _ := VoltsData(testCase.binaryValue, testCase.voltageRange)
					c.So(computedValue, c.ShouldAlmostEqual, testCase.voltage)
				})
			})
		}
	})
}
