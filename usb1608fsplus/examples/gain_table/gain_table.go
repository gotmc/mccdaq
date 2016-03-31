// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package main

import (
	"log"

	"github.com/gotmc/libusb"
	"github.com/gotmc/mccdaq/usb1608fsplus"
)

const millisecondDelay = 100

func main() {
	ctx, err := libusb.Init()
	if err != nil {
		log.Fatal("Couldn't create USB context. Ending now.")
	}
	defer ctx.Exit()

	// Find the first USB fevice with the VendorID and ProductID matching the MCC
	// USB-1608FS-Plus DAQ
	daq, err := usb1608fsplus.GetFirstDevice(ctx)
	if err != nil {
		log.Fatalf("Couldn't find a USB-1608FS-Plus: %s", err)
	}

	// Print some info about the device
	log.Printf("Vendor ID = 0x%x / Product ID = 0x%x\n", daq.DeviceDescriptor.VendorID,
		daq.DeviceDescriptor.ProductID)
	serialNumber, err := daq.SerialNumber()
	log.Printf("Serial number via control transfer = %s", serialNumber)

	// Read the calibration memory to setup the gain table
	gainTable, _ := daq.BuildGainTable()
	for _, inputRange := range usb1608fsplus.InputRanges {
		for ch := 0; ch < 8; ch++ {
			log.Printf("Range = %s Channel = %d Slope = %f Offset = %f", inputRange, ch,
				gainTable.Slope[int(inputRange)][ch], gainTable.Intercept[int(inputRange)][ch])

		}
	}

	daq.Close()
}
