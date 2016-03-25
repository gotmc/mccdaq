// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package main

import (
	"fmt"
	"log"
	"time"

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
	log.Printf("USB ConfigurationIndex = %d\n", daq.ConfigDescriptor.ConfigurationIndex)
	log.Printf("Bulk endpoint address = 0x%x (%b)\n",
		daq.BulkEndpoint.EndpointAddress, daq.BulkEndpoint.EndpointAddress)

	// Test blinking the LED
	numBlinks := 10
	bytesSent, err := daq.BlinkLED(numBlinks)
	if err != nil {
		fmt.Errorf("Error blinking LED %s", err)
	}
	log.Printf("Sent %d byte of data to blink LED %d times.", bytesSent, numBlinks)

	// Close the DAQ
	time.Sleep(millisecondDelay * time.Millisecond)
	daq.Close()
}
