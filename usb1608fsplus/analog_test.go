// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"encoding/json"
	"fmt"
	"log"
	"testing"

	c "github.com/smartystreets/goconvey/convey"
)

type FakeDAQer struct {
	Ranges [8]byte
}

func (f *FakeDAQer) SendCommandToDevice(cmd command, data []byte) (int, error) {
	if cmd == commandAnalogConfig {
		if len(data) != len(f.Ranges) {
			return 0, fmt.Errorf("data is wrong length %d", len(data))
		}
		for i, datum := range data {
			f.Ranges[i] = datum
		}
		return len(data), nil
	}
	return 0, fmt.Errorf("Error sending command to fake DAQer device")
}

func (f *FakeDAQer) ReadCommandFromDevice(cmd command, data []byte) (int, error) {
	if cmd == commandAnalogConfig {
		if len(data) != len(f.Ranges) {
			return 0, fmt.Errorf("data is wrong length %d", len(data))
		}
		for i, x := range f.Ranges {
			data[i] = x
		}
		return len(data), nil
	}
	return 0, fmt.Errorf("Error sending command to fake DAQer device")
}

func (f *FakeDAQer) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (f *FakeDAQer) Status() (byte, error) {
	return 0x0, nil
}

func TestSetScanRanges(t *testing.T) {
	givenRanges := [...]byte{0x0, 0x0, 0x1, 0x1, 0x3, 0x3, 0x5, 0x5}
	f := FakeDAQer{}
	ai := AnalogInput{
		DAQer:             &f,
		Frequency:         500,
		TransferMode:      ImmediateTransfer,
		Trigger:           RisingEdgeTrigger,
		UseExternalPacer:  true,
		OutputPacerOnSync: true,
		DebugMode:         false,
		Stall:             StallInhibited,
	}
	for i, givenRange := range givenRanges {
		ai.Channels[i].Range = VoltageRange(givenRange)
	}
	c.Convey("Given the need to set the scan ranges in the DAQ", t, func() {
		c.Convey("When the SetScanRanges() method is called", func() {
			ai.SetScanRanges()
			c.Convey("Then the ranges should be written to the DAQ", func() {
				ranges, _ := ai.ScanRanges()
				c.So(ranges, c.ShouldResemble, givenRanges[:])
			})
		})
	})
}

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
		preRound      float64
		expectedValue int
	}{
		{0.499, 0},
		{0.4, 0},
		{1.2, 1},
		{799.00, 799},
		{799.90, 800},
		{500000.4, 500000},
	}
	c.Convey("Given the need to round float64 numbers", t, func() {
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

func TestAdjustRawValue(t *testing.T) {
	testCases := []struct {
		value         int
		slope         float64
		offset        float64
		adjustedValue int
	}{
		{0x0000, 1.0, 0.0, 0x0000},
		{0x8000, 1.0, 0.0, 0x8000},
		{0x8000, 1.154856, -5152.185547, 0x7FB2},
		{0x8000, 1.155244, -5451.133301, 0x7E94},
	}
	c.Convey("Given the need to adjust raw binary reading", t, func() {
		for _, testCase := range testCases {
			conveyance := fmt.Sprintf("When a %f slope and %f offset is provided to %#x", testCase.slope, testCase.offset, testCase.value)
			c.Convey(conveyance, func() {
				conveyance := fmt.Sprintf("Then the adjusted value should be %#x", testCase.adjustedValue)
				c.Convey(conveyance, func() {
					computedValue := adjustRawValue(testCase.value, testCase.slope, testCase.offset)
					c.So(computedValue, c.ShouldEqual, testCase.adjustedValue)
				})
			})
		}
	})
}

func TestStallMarshalJSON(t *testing.T) {
	c.Convey("Given the need to marshal Stall into JSON", t, func() {
		c.Convey("When StallOnOverrun is marshaled", func() {
			var s = struct {
				Stall Stall `json:"stall_overrun"`
			}{
				StallOnOverrun,
			}
			c.Convey("Then the JSON object should have {\"stall_overrun\":true}", func() {
				b, err := json.Marshal(&s)
				if err != nil {
					log.Printf("Error marshaling StallOnOverrun: %s", err)
				}
				c.So(b, c.ShouldResemble, []byte(`{"stall_overrun":true}`))
			})
		})
		c.Convey("When StallInhibited is marshaled", func() {
			var s = struct {
				Stall Stall `json:"stall_overrun"`
			}{
				StallInhibited,
			}
			c.Convey("Then the JSON object should have {\"stall_overrun\":false}", func() {
				b, err := json.Marshal(&s)
				if err != nil {
					log.Printf("Error marshaling StallInhibited: %s", err)
				}
				c.So(b, c.ShouldResemble, []byte(`{"stall_overrun":false}`))
			})
		})
	})
}

func TestTransferModeMarshalJSON(t *testing.T) {
	c.Convey("Given the need to marshal TransferMode into JSON", t, func() {
		c.Convey("When BlockTransfer is marshaled", func() {
			var s = struct {
				TransferMode TransferMode `json:"block_transfer"`
			}{
				BlockTransfer,
			}
			c.Convey("Then the JSON object should have {\"block_transfer\":true}", func() {
				b, err := json.Marshal(&s)
				if err != nil {
					log.Printf("Error marshaling BlockTransfer: %s", err)
				}
				c.So(b, c.ShouldResemble, []byte(`{"block_transfer":true}`))
			})
		})
		c.Convey("When ImmediateTransfer is marshaled", func() {
			var s = struct {
				TransferMode TransferMode `json:"block_transfer"`
			}{
				ImmediateTransfer,
			}
			c.Convey("Then the JSON object should have {\"block_transfer\":false}", func() {
				b, err := json.Marshal(&s)
				if err != nil {
					log.Printf("Error marshaling ImmediateTransfer: %s", err)
				}
				c.So(b, c.ShouldResemble, []byte(`{"block_transfer":false}`))
			})
		})
	})
}

func TestTriggerTypeMarshalJSON(t *testing.T) {
	c.Convey("Given the need to marshal TriggerType into JSON", t, func() {
		c.Convey("When NoExternalTrigger is marshaled", func() {
			var s = struct {
				Trigger TriggerType `json:"trigger"`
			}{
				NoExternalTrigger,
			}
			c.Convey("Then the JSON object should have {\"trigger\":\"none\"}", func() {
				b, err := json.Marshal(&s)
				if err != nil {
					log.Printf("Error marshaling NoExternalTrigger: %s", err)
				}
				c.So(b, c.ShouldResemble, []byte(`{"trigger":"none"}`))
			})
		})
	})
}

func TestVoltageRangeMarshalJSON(t *testing.T) {
	c.Convey("Given the need to marshal VoltageRange into JSON", t, func() {
		c.Convey("When Range2V is marshaled", func() {
			var s = struct {
				Range VoltageRange `json:"range"`
			}{
				Range2V,
			}
			c.Convey("Then the JSON object should have {\"range\":\"2V\"}", func() {
				b, err := json.Marshal(&s)
				if err != nil {
					log.Printf("Error marshaling Range2V: %s", err)
				}
				c.So(b, c.ShouldResemble, []byte(`{"range":"2V"}`))
			})
		})
	})
}
