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

	// Create the USB-1608FS-Plus DAQ device
	daq, err := usb1608fsplus.NewViaSN(ctx, "01ACD31D")
	if err != nil {
		log.Fatalf("Something bad getting S/N happened: %s", err)
	}
	// If you just want to grab the first USB-1608FS-Plus that's attached, you
	// can use:
	// daq, err := usb1608fsplus.GetFirstDevice(ctx)

	// Print some info about the device
	log.Printf("Vendor ID = 0x%x / Product ID = 0x%x\n", daq.DeviceDescriptor.VendorID,
		daq.DeviceDescriptor.ProductID)
	serialNumber, err := daq.SerialNumber()
	log.Printf("Serial number via control transfer = %s", serialNumber)
	log.Printf("USB ConfigurationIndex = %d\n", daq.ConfigDescriptor.ConfigurationIndex)
	log.Printf("Bulk endpoint address = 0x%x (%b)\n",
		daq.BulkEndpoint.EndpointAddress, daq.BulkEndpoint.EndpointAddress)

	// Test blinking the LED
	numBlinks := 5
	actualBlinks, err := daq.BlinkLED(numBlinks)
	if err != nil {
		fmt.Errorf("Error blinking LED %s", err)
	}
	log.Printf("Sent %d byte of data to blink LED %d times.", actualBlinks, numBlinks)

	// Get status
	status, err := daq.Status()
	log.Printf("Status = %v", status)

	// Read the calibration memory to setup the gain table
	gainTable, _ := daq.BuildGainTable()
	log.Printf("Slope = %v\n", gainTable.Slope)
	log.Printf("Intercept = %v\n", gainTable.Intercept)

	/**************************
	* Start the Analog Scan   *
	**************************/

	// Setup stuff
	splitScansIn := 2
	totalScans := 512
	scansPerRead := totalScans / splitScansIn
	var frequency float64 = 20000.0

	// Create new analog input and ensure the scan is stopped and buffer cleared
	ai := daq.NewAnalogInput(frequency)
	ai.StopScan()
	time.Sleep(time.Second)
	ai.ClearScanBuffer()
	// Setup the analog input scan
	ai.TransferMode = usb1608fsplus.BlockTransfer
	ai.DebugMode = true
	ai.ConfigureChannel(0, true, 5, "Vin1")
	ai.SetScanRanges()
	// Read the scan ranges
	time.Sleep(time.Second)
	blah, err := ai.ScanRanges()
	log.Printf("Ranges = %v\n", blah)

	// Start the scan
	ai.StartScan(totalScans)
	for j := 0; j < splitScansIn; j++ {
		time.Sleep(1 * time.Second)
		data, err := ai.ReadScan(scansPerRead)
		if err != nil {
			log.Fatalf("Error reading scan: %s", err)
		}
		for i := 0; i < 8; i += 2 {
			log.Printf("data[%d:%d] = %d %d\n", i, i+1, data[i+1], data[i])
		}
		for i := scansPerRead - 8; i < scansPerRead; i += 2 {
			log.Printf("data[%d:%d] = %d %d\n", i, i+1, data[i+1], data[i])
		}
		log.Printf("data is %d bytes\n", len(data))
	}
	ai.StopScan()
	time.Sleep(1 * time.Second)

	daq.Close()

}
