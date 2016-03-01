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
	// usb1608fsplus.Reset(dh)
	usb1608fsplus.BlinkLED(dh, 3)

	err = dh.ReleaseInterface(0)
	if err != nil {
		log.Printf("Error releasing interface %s", err)
	}
	dh.Close()
}
