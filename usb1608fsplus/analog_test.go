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
		frequency float64
		channels  byte
		options   byte
		packet    []byte
	}{
		{1, 0.00, 0x00, 0x00, []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0}},
		{1, 10000.0, 0x01, 0x00, []byte{01, 0, 00, 00, 159, 15, 00, 00, 1, 0}},
		{256, 50000.0, 0xff, 0xff, []byte{0, 1, 0, 0, 31, 3, 0, 0, 255, 255}},
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

func TestCalculatingPacerPeriod(t *testing.T) {
	testCases := []struct {
		frequency   float64
		pacerPeriod int
	}{
		{40e6, 79},
		{10000.0, 3999},
		{50000.0, 799},
	}
	c.Convey("Given the need to calculate the pacer period", t, func() {
		for _, testCase := range testCases {
			conveyance := fmt.Sprintf("When the frequency is %f Hz", testCase.frequency)
			c.Convey(conveyance, func() {
				conveyance := fmt.Sprintf("Then the pacer period should be %d", testCase.pacerPeriod)
				c.Convey(conveyance, func() {
					c.So(calculatePacerPeriod(testCase.frequency), c.ShouldEqual, testCase.pacerPeriod)
				})
			})
		}
	})
}

func TestRound(t *testing.T) {
	testCases := []struct {
		preRound      float32
		expectedValue int
	}{
		{0.499, 0},
		{0.4, 0},
		{1.2, 1},
		{799.00, 799},
		{799.90, 800},
		{500000.4, 500000},
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
