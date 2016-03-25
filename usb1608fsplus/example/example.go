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

	// Create new analog input and ensure the scan is stopped and buffer cleared
	var frequency float64 = 10000.0
	ai := daq.NewAnalogInput(frequency)
	ai.StopScan()
	time.Sleep(millisecondDelay * time.Millisecond)
	ai.ClearScanBuffer()

	// Setup the analog input scan
	ai.TransferMode = usb1608fsplus.BlockTransfer
	ai.DebugMode = true
	ai.ConfigureEnableChannel(0, "5V", "Vin1")
	ai.ConfigureEnableChannel(1, "5V", "Vin2")
	ai.ConfigureEnableChannel(2, "10V", "Vin3")
	ai.ConfigureEnableChannel(3, "10V", "Vin4")
	ai.ConfigureEnableChannel(4, "1V", "Iin1")
	ai.ConfigureEnableChannel(5, "1V", "Iin2")
	ai.ConfigureEnableChannel(6, "2V", "Iin3")
	ai.ConfigureEnableChannel(7, "2V", "Iin4")
	ai.SetScanRanges()

	// Read the scan ranges
	time.Sleep(millisecondDelay * time.Millisecond)
	scanRanges, err := ai.ScanRanges()
	log.Printf("Ranges = %v\n", scanRanges)

	// Read the totalScans using splitScansIn number of scans
	const (
		scansPerBuffer = 256
		totalBuffers   = 10
	)
	expectedDuration := (scansPerBuffer * totalBuffers) / frequency
	ai.StartScan(0)
	start := time.Now()
	totalBytesRead := 0
	for j := 0; j < totalBuffers; j++ {
		// time.Sleep(millisecondDelay * time.Millisecond)
		data, err := ai.ReadScan(scansPerBuffer)
		totalBytesRead += len(data)
		if err != nil {
			// Stop the analog scan and close the DAQ
			ai.StopScan()
			time.Sleep(millisecondDelay * time.Millisecond)
			daq.Close()
			log.Fatalf("Error reading scan: %s", err)
		}
		// Print the first 8 bytes and the last 8 bytes of each read
		bytesToShow := 8
		for i := 0; i < bytesToShow; i += 2 {
			log.Printf("data[%d:%d] = 0x%02x%02x\n", i, i+1, data[i+1], data[i])
		}
		for i := len(data) - bytesToShow; i < len(data); i += 2 {
			log.Printf("data[%d:%d] = 0x%02x%02x\n", i, i+1, data[i+1], data[i])
		}
		log.Printf("Length of data is %d bytes\n", len(data))
	}
	elapsed := time.Since(start)
	log.Printf("Reading %d bytes took %.2f s", totalBytesRead, elapsed.Seconds())
	log.Printf("Anticipated reading %d bytes to take %.2f s",
		scansPerBuffer*totalBuffers*ai.NumEnabledChannels()*2, expectedDuration)
	// Stop the analog scan and close the DAQ
	time.Sleep(millisecondDelay * time.Millisecond)
	ai.StopScan()
	time.Sleep(millisecondDelay * time.Millisecond)
	daq.Close()
}
