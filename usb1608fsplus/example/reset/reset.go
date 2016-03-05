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

func main() {
	// Create the USB context
	ctx, err := libusb.Init()
	if err != nil {
		log.Fatal("Couldn't create USB context. Ending now.")
	}
	defer ctx.Exit()

	// Get the USB device and handle
	_, dh, _ := ctx.OpenDeviceWithVendorProduct(0x09db, 0x00ea)
	defer dh.Close()

	// Reset the device
	usb1608fsplus.StopAnalogScan(dh)
	time.Sleep(time.Second)
	usb1608fsplus.ClearScanBuffer(dh)
	usb1608fsplus.Reset(dh)
	time.Sleep(2 * time.Second)
}
