// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package main

import (
	"fmt"
	"log"

	"github.com/gotmc/libusb"
	"github.com/gotmc/mccdaq/usb1608fsplus"
)

func main() {
	ctx, err := libusb.Init()
	if err != nil {
		log.Fatal("Couldn't create USB context. Ending now.")
	}
	defer ctx.Exit()

	dev, dh, _ := ctx.OpenDeviceWithVendorProduct(0x09db, 0x00ea)
	description, _ := dev.GetDeviceDescriptor()
	log.Printf("Product = 0x%x\n", description.ProductID)

	err = dh.ClaimInterface(0)
	if err != nil {
		log.Printf("Error claiming interface %s", err)
	}

	// Test blinking the LED
	blinks := 2
	count, err := usb1608fsplus.BlinkLED(dh, blinks)
	if err != nil {
		fmt.Errorf("Error blinking LED %s", err)
	}
	log.Printf("Sent %d data to blink LED %d times.", count, blinks)

	// Get status
	status, err := usb1608fsplus.Status(dh)
	log.Printf("Status = %v", status)

	// Get serial number via control transfer
	serialNumber, err := usb1608fsplus.SerialNumber(dh)
	log.Printf("Serial number via control transfer = %s", serialNumber)
	desc, _ := dev.GetDeviceDescriptor()
	sn, _ := dh.GetStringDescriptorASCII(desc.SerialNumberIndex)
	log.Printf("Serial number via libusb device descriptor = %s\n", sn)
	log.Printf("Vendor ID via libusb device descriptor = 0x%x\n", desc.VendorID)
	log.Printf("Product ID via libusb device descriptor = 0x%x\n", desc.ProductID)

	// Read the calibration memory to setup the gain table
	gainTable, _ := usb1608fsplus.BuildGainTable(dh)
	log.Printf("Slope = %v\n", gainTable.Slope)
	log.Printf("Intercept = %v\n", gainTable.Intercept)

	// Release the interface and close up shop
	err = dh.ReleaseInterface(0)
	if err != nil {
		log.Printf("Error releasing interface %s", err)
	}
	dh.Close()
}
