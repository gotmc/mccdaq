// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"fmt"
	"testing"

	c "github.com/smartystreets/goconvey/convey"
)

func TestValidCalMemoryRange(t *testing.T) {
	testCases := []struct {
		address int
		count   int
		valid   bool
	}{
		{0, 0, false},
		{-1, 1, false},
		{0, 1, true},
		{0, 768, true},
		{0, 769, false},
		{1, 767, true},
		{1, 768, false},
		{0x2ff, 1, true},
		{0x2ff, 2, false},
	}
	c.Convey("Given the need to validate the calibration memory range", t, func() {
		for _, testCase := range testCases {
			bytePlurality := "bytes"
			if testCase.count == 1 {
				bytePlurality = "byte"
			}
			conveyance := fmt.Sprintf(
				"When reading %d %s starting at address %x",
				testCase.count,
				bytePlurality,
				testCase.address,
			)
			c.Convey(conveyance, func() {
				validity := "invalid"
				if testCase.valid {
					validity = "valid"
				}
				conveyance := fmt.Sprintf("Then the cal memory range is %s", validity)
				c.Convey(conveyance, func() {
					computedValue := validCalMemoryRange(testCase.address, testCase.count)
					c.So(computedValue, c.ShouldResemble, testCase.valid)
				})
			})
		}
	})
}
