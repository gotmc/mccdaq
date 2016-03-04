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

func TestPackScanData(t *testing.T) {
	testCases := []struct {
		numScans  int
		frequency float32
		channels  byte
		options   byte
		packet    []byte
	}{
		{1, 0.00, 0x00, 0x00, []byte{01, 00, 00, 00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}},
		{1, 1.00, 0x00, 0x00, []byte{01, 00, 00, 00, 0x80, 0x96, 0x18, 0x4c, 0x00, 0x00}},
		{100, 10.0e6, 0xff, 0xff, []byte{100, 00, 00, 00, 0x00, 0x00, 0x9e, 0x42, 0xff, 0xff}},
		{512, 20.0e3, 0xaa, 0xbb, []byte{00, 02, 00, 00, 0x00, 0xe0, 0xf9, 0x44, 0xaa, 0xbb}},
	}
	c.Convey("Given the need to create the scan data packet", t, func() {
		for _, testCase := range testCases {
			scanText := "scans"
			if testCase.numScans == 1 {
				scanText = "scan"
			}
			frequency := testCase.frequency
			if frequency > maxFrequency {
				frequency = maxFrequency
			}
			conveyance := fmt.Sprintf(
				"When there's %d %s at %g Hz for 0x%x channels & 0x%x options",
				testCase.numScans,
				scanText,
				frequency,
				testCase.channels,
				testCase.options,
			)
			c.Convey(conveyance, func() {
				conveyance := fmt.Sprintf("Then the data packet should be %X", testCase.packet)
				c.Convey(conveyance, func() {
					computedValue := packScanData(
						testCase.numScans,
						testCase.frequency,
						testCase.channels,
						testCase.options,
					)
					c.So(computedValue, c.ShouldResemble, testCase.packet)
				})
			})
		}
	})
}

func TestRound(t *testing.T) {
	testCases := []struct {
		preRound      float32
		expectedValue float32
	}{
		{1.2, 1.0},
		{500000.4, 500000.0},
	}
	c.Convey("Given the need to round float32 numbers", t, func() {
		for _, testCase := range testCases {
			conveyance := fmt.Sprintf("When %f is provided to the round() function", testCase.preRound)
			c.Convey(conveyance, func() {
				conveyance := fmt.Sprintf("Then the result should be %f", testCase.expectedValue)
				c.Convey(conveyance, func() {
					computedValue := round(testCase.preRound)
					c.So(computedValue, c.ShouldEqual, testCase.expectedValue)
				})
			})
		}
	})
}
