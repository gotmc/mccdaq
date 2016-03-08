// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"fmt"
	"time"

	"github.com/gotmc/libusb"
)

const (
	vendorID       = 0x09db
	productID      = 0x00ea
	defaultTimeout = 2000
)

type USB1608FSPlus struct {
	Timeout          int
	Device           *libusb.Device
	DeviceDescriptor *libusb.DeviceDescriptor
	DeviceHandle     *libusb.DeviceHandle
	ConfigDescriptor *libusb.ConfigDescriptor
	BulkEndpoint     *libusb.EndpointDescriptor
}

// Create creates a new instance of a USB1608FSPlus.
func Create(ctx *libusb.Context) (*USB1608FSPlus, error) {
	var daq USB1608FSPlus
	dev, dh, err := ctx.OpenDeviceWithVendorProduct(vendorID, productID)
	if err != nil {
		return &daq, fmt.Errorf("Error opening the USB-1608FS-Plus using the VendorID and ProductID, %s", err)
	}
	err = dh.ClaimInterface(0)
	if err != nil {
		return &daq, fmt.Errorf("Error claiming the bulk interface %s", err)
	}
	daq.Timeout = defaultTimeout
	daq.Device = dev
	daq.DeviceHandle = dh
	deviceDescriptor, err := daq.Device.GetDeviceDescriptor()
	if err != nil {
		return &daq, fmt.Errorf("Error getting device descriptor %s", err)
	}
	daq.DeviceDescriptor = deviceDescriptor
	configDescriptor, err := daq.Device.GetActiveConfigDescriptor()
	if err != nil {
		return &daq, fmt.Errorf("Error getting active config descriptor. %s", err)
	}
	daq.ConfigDescriptor = configDescriptor
	firstDescriptor := configDescriptor.SupportedInterfaces[0].InterfaceDescriptors[0]
	daq.BulkEndpoint = firstDescriptor.EndpointDescriptors[0]
	return &daq, nil
}

func (daq *USB1608FSPlus) Close() error {
	// Release the interface and close up shop
	err := daq.DeviceHandle.ReleaseInterface(0)
	if err != nil {
		return fmt.Errorf("Error releasing interface %s", err)
	}
	time.Sleep(1 * time.Second)
	_, err = daq.Reset()
	if err != nil {
		return fmt.Errorf("Error reseting USB-1608FS-Plus %s", err)
	}
	time.Sleep(1 * time.Second)
	daq.DeviceHandle.Close()
	return nil
}

// Reset resets the device.
func (daq *USB1608FSPlus) Reset() (int, error) {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	ret, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandReset), 0x0, 0x0, []byte{0x00}, 1, timeout)
	if err != nil {
		return ret, fmt.Errorf("Error resetting devices %s", err)
	}
	return ret, nil
}
