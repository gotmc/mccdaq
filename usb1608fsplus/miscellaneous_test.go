// Copyright (c) 2016-2017 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"fmt"
	"testing"
)

func TestConvertBinaryValueToVoltage(t *testing.T) {
	testCases := []struct {
		expected float64
		vr       VoltageRange
		given    []byte
	}{
		{-10.0, Range10V, []byte{0, 0}},
		{0.0, Range10V, []byte{0x00, 0x80}},
		{9.99969482421875, Range10V, []byte{0xff, 0xff}},
		{-5.0, Range5V, []byte{0, 0}},
		{0.0, Range5V, []byte{0x00, 0x80}},
		{4.999847412109375, Range5V, []byte{0xff, 0xff}},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Convert binary value %#x", tc.given), func(t *testing.T) {
			t.Parallel()
			computed, _ := RawVoltsFromWord(tc.given, tc.vr)
			if computed != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, computed)
			}
		})
	}
}

func TestBadConvertBinaryValueToVoltage(t *testing.T) {
	testCases := []struct {
		given    []byte
		vr       VoltageRange
		expected string
	}{
		{[]byte{0, 0, 0}, Range5V, "binary value must be 2 bytes"},
		{[]byte{0}, Range5V, "binary value must be 2 bytes"},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Convert binary value %#x", tc.given), func(t *testing.T) {
			// t.Parallel()
			_, err := RawVoltsFromWord(tc.given, tc.vr)
			if err.Error() != tc.expected {
				t.Errorf("Expected `%v`, got `%v`", tc.expected, err)
			}
		})
	}
}

func TestVoltsFromWord(t *testing.T) {
	testCases := []struct {
		given    []byte
		vr       VoltageRange
		slope    float64
		offset   float64
		expected float64
	}{
		{[]byte{0x00, 0x00}, Range10V, 1.0, 0.0, -10.0},
		{[]byte{0x00, 0x00}, Range10V, 2.0, 0.0, -10.0},
		{[]byte{0xff, 0xff}, Range10V, 1.0, 0.0, 9.99969482421875},
		{[]byte{0x00, 0x80}, Range10V, 1.0, 0.0, 0.00},
		{[]byte{0xFF, 0x7F}, Range10V, 1.0, 1.0, 0.00},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Volts from %#x", tc.given), func(t *testing.T) {
			t.Parallel()
			computed, _ := VoltsFromWord(tc.given, tc.vr, tc.slope, tc.offset)
			if computed != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, computed)
			}
		})
	}
}

func TestAdjustedRawValue(t *testing.T) {
	testCases := []struct {
		given    uint16
		slope    float64
		offset   float64
		expected int
	}{
		{0, 1.0, 0.0, 0},
		{0, 1.0, 1.0, 1},
		{10, 1.0, 1.0, 11},
		{10, 2.0, 0.0, 20},
		{10, 2.0, -1.0, 19},
		{65535, 1.00, -5000, 60535},
		{65535, 1.15, -5000, 70365},
	}
	for _, tc := range testCases {
		testName := fmt.Sprintf("value %v slope %v offset %v", tc.given, tc.slope, tc.offset)
		t.Run(testName, func(t *testing.T) {
			// t.Parallel()
			computed := adjustRawValue(tc.given, tc.slope, tc.offset)
			if computed != tc.expected {
				t.Errorf("Expected %v, got %v", tc.expected, computed)
			}
		})
	}
}
