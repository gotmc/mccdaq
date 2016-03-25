// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package main

import (
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

	// Create the USB-1608FS-Plus DAQ device
	daq, err := usb1608fsplus.NewViaSN(ctx, "01ACD31D")
	if err != nil {
		log.Fatalf("Something bad getting S/N happened: %s", err)
	}

	// Print some info about the device
	log.Printf("Vendor ID = 0x%x / Product ID = 0x%x\n", daq.DeviceDescriptor.VendorID,
		daq.DeviceDescriptor.ProductID)
	serialNumber, err := daq.SerialNumber()
	log.Printf("Serial number via control transfer = %s", serialNumber)
	log.Printf("USB ConfigurationIndex = %d\n", daq.ConfigDescriptor.ConfigurationIndex)
	log.Printf("Bulk endpoint address = 0x%x (%b)\n",
		daq.BulkEndpoint.EndpointAddress, daq.BulkEndpoint.EndpointAddress)

	// Get status
	status, err := daq.Status()
	log.Printf("Status = %v", status)

	// Close the DAQ
	time.Sleep(millisecondDelay * time.Millisecond)
	daq.Close()
}
