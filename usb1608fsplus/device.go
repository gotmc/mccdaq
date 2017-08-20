// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"fmt"
	"log"
	"time"

	"github.com/gotmc/libusb"
)

const (
	vendorID       = 0x09db
	productID      = 0x00ea
	defaultTimeout = 2000
	msSleepTime    = 500
)

// DAQer defines the interface required for a DAQ.
type DAQer interface {
	SendCommandToDevice(cmd command, data []byte) (int, error)
	ReadCommandFromDevice(cmd command, data []byte) (int, error)
	Read(p []byte) (n int, err error)
	Status() (byte, error)
}

// USB1608fsplus models the USB-1608FS-Plus DAQ.
type USB1608fsplus struct {
	Timeout          int
	Device           *libusb.Device
	DeviceDescriptor *libusb.DeviceDescriptor
	DeviceHandle     *libusb.DeviceHandle
	ConfigDescriptor *libusb.ConfigDescriptor
	BulkEndpoint     *libusb.EndpointDescriptor
}

// Init intializes a new libusb session/context by creating a new Context and
// returning a pointer to that Context.
func Init() (*libusb.Context, error) {
	return libusb.NewContext()
}

// NewViaSN creates a new daq instance by searching through the list of USB
// devices for the given serial number.
func NewViaSN(ctx *libusb.Context, sn string) (*USB1608fsplus, error) {
	var daq USB1608fsplus
	usbDevices, err := ctx.GetDeviceList()
	if err != nil {
		return &daq, fmt.Errorf("Error getting USB device list: %s", err)
	}
	// Search through the USB devices looking for serial number
	for _, usbDevice := range usbDevices {
		usbDeviceDescriptor, err := usbDevice.GetDeviceDescriptor()
		if err != nil {
			return &daq, fmt.Errorf("Error getting device descriptor: %s", err)
		}
		// Check the VendorID and Product ID. If those don't equate to MCC and
		// USB-1608FS-Plus, then there's no reason to open the device and read its
		// S/N.
		if usbDeviceDescriptor.VendorID == vendorID &&
			usbDeviceDescriptor.ProductID == productID {
			// Found a USB-1608FS-Plus
			usbDeviceHandle, err := usbDevice.Open()
			if err != nil {
				return &daq, fmt.Errorf("Error getting device handle: %s", err)
			}
			serialNum, err := usbDeviceHandle.GetStringDescriptorASCII(
				usbDeviceDescriptor.SerialNumberIndex)
			if err != nil {
				return &daq, fmt.Errorf("Error reading S/N: %s", err)
			}
			if serialNum == sn {
				log.Printf("Found S/N %s. Creating device", sn)
				return create(usbDevice, usbDeviceHandle)
			}
			usbDeviceHandle.Close()
		}
	}
	// Close the list of devices
	return &daq, fmt.Errorf("couldn't find daq %s", sn)
}

// GetFirstDevice creates a new instance of a daq using the first
// USB-1608FS-Plus found in the USB context.
func GetFirstDevice(ctx *libusb.Context) (*USB1608fsplus, error) {
	var daq USB1608fsplus
	dev, dh, err := ctx.OpenDeviceWithVendorProduct(vendorID, productID)
	if err != nil {
		return &daq, fmt.Errorf("error opening the daq, %s", err)
	}
	return create(dev, dh)
}

func create(dev *libusb.Device, dh *libusb.DeviceHandle) (*USB1608fsplus, error) {
	var daq USB1608fsplus
	err := dh.ClaimInterface(0)
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

// Close implements the Closer interface for USB1608fsplus
func (daq *USB1608fsplus) Close() error {
	// Release the interface and close up shop
	err := daq.DeviceHandle.ReleaseInterface(0)
	if err != nil {
		return fmt.Errorf("Error releasing interface %s", err)
	}
	time.Sleep(msSleepTime * time.Millisecond)
	_, err = daq.Reset()
	if err != nil {
		return fmt.Errorf("Error reseting USB-1608FS-Plus %s", err)
	}
	time.Sleep(msSleepTime * time.Millisecond)
	daq.DeviceHandle.Close()
	return nil
}

// Reset resets the device.
func (daq *USB1608fsplus) Reset() (int, error) {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	ret, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandReset), 0x0, 0x0, []byte{0x00}, 1, daq.Timeout)
	if err != nil {
		return ret, fmt.Errorf("Error resetting devices %s", err)
	}
	return ret, nil
}

// SendCommandToDevice sends the given command and data to the device and
// returns the number of bytes received and whether or not an error was
// received.
func (daq *USB1608fsplus) SendCommandToDevice(cmd command, data []byte) (int, error) {
	if data == nil {
		data = []byte{0}
	}
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	bytesReceived, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(cmd), 0x0, 0x0, data, len(data), daq.Timeout)
	if err != nil {
		return bytesReceived, fmt.Errorf("error sending command '%s' to device: %s", cmd, err)
	}
	return bytesReceived, nil
}

// ReadCommandFromDevice sends a command to the DAQ via USB and reads the
// results of the command.
func (daq *USB1608fsplus) ReadCommandFromDevice(cmd command, data []byte) (int, error) {
	if data == nil {
		data = []byte{0}
	}
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	bytesReceived, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(cmd), 0x0, 0x0, data, len(data), daq.Timeout)
	if err != nil {
		return bytesReceived, fmt.Errorf("error reading command '%s' from device: %s", cmd, err)
	}
	return bytesReceived, nil
}

// Read reads the data using a bulk USB transfer.
func (daq *USB1608fsplus) Read(p []byte) (n int, err error) {
	return daq.DeviceHandle.BulkTransfer(
		daq.BulkEndpoint.EndpointAddress,
		p,
		len(p),
		daq.Timeout,
	)
}
