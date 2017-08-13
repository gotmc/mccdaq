// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package main

import (
	"flag"
	"log"
	"time"

	"github.com/gotmc/libusb"
	"github.com/gotmc/mccdaq/usb1608fsplus"
)

const millisecondDelay = 100

var (
	sn1 string
	sn2 string
)

func init() {
	flag.StringVar(&sn1, "sn1", "01AF3FAE", "MCC DAQ #1's S/N")
	flag.StringVar(&sn2, "sn2", "01AF3FBA", "MCC DAQ #2's S/N")
}

func main() {

	// Parse the config flags
	flag.Parse()
	if sn1 == sn2 {
		log.Fatalf("S/Ns cannot be the same %s.", sn1)
	}
	log.Printf("Looking for S/Ns %s and %s", sn1, sn2)

	// Setup the USB context
	ctx, err := libusb.NewContext()
	if err != nil {
		log.Fatal("Couldn't create USB context. Ending now.")
	}
	defer ctx.Close()

	// Create the USB-1608FS-Plus DAQ devices
	daq1, err := usb1608fsplus.NewViaSN(ctx, sn1)
	if err != nil {
		log.Fatalf("Something bad getting S/N %s happened: %s", sn1, err)
	}
	daq2, err := usb1608fsplus.NewViaSN(ctx, sn2)
	if err != nil {
		log.Fatalf("Something bad getting S/N %s happened: %s", sn2, err)
	}

	// Print some info about each device
	log.Printf("Vendor ID = 0x%x / Product ID = 0x%x\n", daq1.DeviceDescriptor.VendorID,
		daq1.DeviceDescriptor.ProductID)
	serialNumber, err := daq1.SerialNumber()
	log.Printf("Serial number via control transfer = %s", serialNumber)
	log.Printf("USB ConfigurationIndex = %d\n", daq1.ConfigDescriptor.ConfigurationIndex)
	log.Printf("Bulk endpoint address = 0x%x (%b)\n",
		daq1.BulkEndpoint.EndpointAddress, daq1.BulkEndpoint.EndpointAddress)

	// Print some info about each device
	log.Printf("Vendor ID = 0x%x / Product ID = 0x%x\n", daq2.DeviceDescriptor.VendorID,
		daq2.DeviceDescriptor.ProductID)
	serialNumber, err = daq2.SerialNumber()
	log.Printf("Serial number via control transfer = %s", serialNumber)
	log.Printf("USB ConfigurationIndex = %d\n", daq2.ConfigDescriptor.ConfigurationIndex)
	log.Printf("Bulk endpoint address = 0x%x (%b)\n",
		daq2.BulkEndpoint.EndpointAddress, daq2.BulkEndpoint.EndpointAddress)

	// Get status
	status, err := daq1.Status()
	log.Printf("Status = %v", status)

	// Get status
	status, err = daq2.Status()
	log.Printf("Status = %v", status)

	// Close the DAQ
	time.Sleep(millisecondDelay * time.Millisecond)
	daq1.Close()

	// Close the DAQ
	time.Sleep(millisecondDelay * time.Millisecond)
	daq2.Close()

}
