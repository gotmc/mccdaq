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

func main() {
	ctx, err := libusb.Init()
	if err != nil {
		log.Fatal("Couldn't create USB context. Ending now.")
	}
	defer ctx.Exit()

	dev, dh, err := ctx.OpenDeviceWithVendorProduct(0x09db, 0x00ea)
	if err != nil {
		log.Fatalf("Error opening device %s", err)
	}
	description, err := dev.GetDeviceDescriptor()
	if err != nil {
		log.Fatalf("Error getting device descriptor %s", err)
	}
	log.Printf("Product = 0x%x\n", description.ProductID)

	// Claim the interface before starting to communicate
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

	// Get descriptor
	configDescriptor, err := dev.GetActiveConfigDescriptor()
	if err != nil {
		log.Fatalf("Error getting active config descriptor. %s", err)
	}
	fmt.Printf("USB ConfigurationIndex = %d\n", configDescriptor.ConfigurationIndex)
	firstDescriptor := configDescriptor.SupportedInterfaces[0].InterfaceDescriptors[0]

	for i, endpoint := range firstDescriptor.EndpointDescriptors {
		fmt.Printf(
			"   => Endpoint index %d on Interface %d has the following properties:\n",
			i, firstDescriptor.InterfaceNumber)
		fmt.Printf("     => Address: %d (b%08b)\n", endpoint.EndpointAddress, endpoint.EndpointAddress)
		fmt.Printf("       => Endpoint #: %d\n", endpoint.Number())
		fmt.Printf("       => Direction: %s (%d)\n", endpoint.Direction(), endpoint.Direction())
		fmt.Printf("     => Attributes: %d (b%08b) \n", endpoint.Attributes, endpoint.Attributes)
		fmt.Printf("       => Transfer Type: %s (%d) \n", endpoint.TransferType(), endpoint.TransferType())
		fmt.Printf("     => Max packet size: %d\n", endpoint.MaxPacketSize)
	}

	// Grab bulk endpoint
	ep := firstDescriptor.EndpointDescriptors[0]
	log.Printf("Endpoint address = 0x%x (%b)\n", ep.EndpointAddress, ep.EndpointAddress)

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

	// Read one analog reading.
	usb1608fsplus.StopAnalogScan(dh)
	time.Sleep(time.Second)
	foo, err := usb1608fsplus.ReadAnalogInput(dh, 1, 0)
	if err != nil {
		log.Fatalf("Error reading one analog input. %s", err)
	}
	log.Printf("Read analog input %d", foo)

	/**************************
	* Start the Analog Scan   *
	**************************/

	// Setup stuff
	var ranges = make([]byte, 8) // Range 0 is ±10V
	for i := 0; i < len(ranges); i++ {
		ranges[i] = 0
	}
	count = 256
	var channels byte = 0x01 // one bit for each channel
	var frequency float64 = 20000.0
	// options := byte(0x1 | 0x2) // immediate w/ internal pacer on
	options := byte(0x0 | 0x2 | 0x20) // bulk w/ internal pacer on
	numChannels := 1

	// Stop, clear, and configure
	usb1608fsplus.StopAnalogScan(dh)
	time.Sleep(time.Second)
	usb1608fsplus.ClearScanBuffer(dh)
	usb1608fsplus.ConfigAnalogScan(dh, ranges)
	time.Sleep(2 * time.Second)
	blah, err := usb1608fsplus.ReadScanRanges(dh)
	log.Printf("Ranges = %v\n", blah)

	// Start the scan
	usb1608fsplus.StartAnalogScan(dh, count, frequency, channels, options)
	time.Sleep(1 * time.Second)
	data, err := usb1608fsplus.ReadScan(dh, ep, count, numChannels, options)
	for i := 0; i < 16; i++ {
		fmt.Printf("data[%d] = %d\n", i, data[i])
	}
	usb1608fsplus.StopAnalogScan(dh)
	time.Sleep(1 * time.Second)

	// Release the interface and close up shop
	err = dh.ReleaseInterface(0)
	if err != nil {
		log.Printf("Error releasing interface %s", err)
	}
	time.Sleep(1 * time.Second)
	usb1608fsplus.Reset(dh)
	time.Sleep(1 * time.Second)
	dh.Close()
}
