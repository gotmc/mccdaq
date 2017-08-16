// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb20x

import (
	"fmt"
	"log"
	"time"

	"github.com/gotmc/libusb"
)

const (
	vendorID       = 0x09db
	defaultTimeout = 2000
	msSleepTime    = 500
)

// FIXME(mdr): I feel like these should be their own type.
const (
	usb201PID = 0x0113
	usb202PID = 0x012b
	usb204PID = 0x0114
	usb205PID = 0x012c
)

// DAQer defines the interface required for a DAQ.
type DAQer interface {
	SendCommandToDevice(cmd command, data []byte) (int, error)
	ReadCommandFromDevice(cmd command, data []byte) (int, error)
	Read(p []byte) (n int, err error)
	Status() (byte, error)
}

type usb20x struct {
	Timeout          int
	Device           *libusb.Device
	DeviceDescriptor *libusb.DeviceDescriptor
	DeviceHandle     *libusb.DeviceHandle
	ConfigDescriptor *libusb.ConfigDescriptor
	BulkEndpoint     *libusb.EndpointDescriptor
}

// NewViaSN creates a new daq instance by searching through the list of USB
// devices for the given serial number.
func NewViaSN(ctx *libusb.Context, sn string) (*usb20x, error) {
	var daq usb20x
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
		// Check the VendorID and Product ID. If those don't equate to MCC and one
		// of the USB20X Product IDs, then there's no reason to open the device and
		// read its S/N.
		if usbDeviceDescriptor.VendorID == vendorID &&
			(usbDeviceDescriptor.ProductID == usb201PID ||
				usbDeviceDescriptor.ProductID == usb202PID ||
				usbDeviceDescriptor.ProductID == usb204PID ||
				usbDeviceDescriptor.ProductID == usb205PID) {
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
	return &daq, fmt.Errorf("couldn't find device s/n %s", sn)
}

// GetFirstUSB201 creates a new instance of a daq using the first
// USB-201 found in the USB context.
func getFirstUSB201(ctx *libusb.Context) (*usb20x, error) {
	return getFirstDevice(ctx, usb201PID)
}

// GetFirstUSB202 creates a new instance of a daq using the first
// USB-202 found in the USB context.
func getFirstUSB202(ctx *libusb.Context) (*usb20x, error) {
	return getFirstDevice(ctx, usb202PID)
}

// GetFirstUSB204 creates a new instance of a daq using the first
// USB-204 found in the USB context.
func getFirstUSB204(ctx *libusb.Context) (*usb20x, error) {
	return getFirstDevice(ctx, usb204PID)
}

// GetFirstUSB205 creates a new instance of a daq using the first
// USB-205 found in the USB context.
func getFirstUSB205(ctx *libusb.Context) (*usb20x, error) {
	return getFirstDevice(ctx, usb205PID)
}

func getFirstDevice(ctx *libusb.Context, productID uint) (*usb20x, error) {
	var daq usb20x
	dev, dh, err := ctx.OpenDeviceWithVendorProduct(vendorID, uint16(productID))
	if err != nil {
		return &daq, fmt.Errorf("Error opening the USB-1608FS-Plus using the VendorID and ProductID, %s", err)
	}
	return create(dev, dh)
}

func create(dev *libusb.Device, dh *libusb.DeviceHandle) (*usb20x, error) {
	var daq usb20x
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

func (daq *usb20x) Close() error {
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
func (daq *usb20x) Reset() (int, error) {
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
func (daq *usb20x) SendCommandToDevice(cmd command, data []byte) (int, error) {
	if data == nil {
		data = []byte{0}
	}
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	bytesReceived, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(cmd), 0x0, 0x0, data, len(data), daq.Timeout)
	if err != nil {
		return bytesReceived, fmt.Errorf("Error sending command '%s' to device: %s", cmd, err)
	}
	return bytesReceived, nil
}

func (daq *usb20x) ReadCommandFromDevice(cmd command, data []byte) (int, error) {
	if data == nil {
		data = []byte{0}
	}
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	bytesReceived, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(cmd), 0x0, 0x0, data, len(data), daq.Timeout)
	if err != nil {
		return bytesReceived, fmt.Errorf("Error reading command '%s' from device: %s", cmd, err)
	}
	return bytesReceived, nil
}

func (daq *usb20x) Read(p []byte) (n int, err error) {
	return daq.DeviceHandle.BulkTransfer(
		daq.BulkEndpoint.EndpointAddress,
		p,
		len(p),
		daq.Timeout,
	)
}
