// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/gotmc/libusb"
)

func byteSlice(i int) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(i))
	return b
}

// BlinkLED blinks the LED the given number of times. Note, the LED starts
// being unlit, but will end being lit.
func BlinkLED(dh *libusb.DeviceHandle, count int) (int, error) {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	// data := byteSlice(count)
	data := make([]byte, 1)
	data[0] = byte(count)

	ret, err := dh.ControlTransfer(
		requestType, byte(commandBlinkLED), 0x0, 0x0, data, len(data), timeout)
	if err != nil {
		return ret, fmt.Errorf("Error blinking LED %s", err)
	}
	return ret, nil
}

// Reset resets the device.
func Reset(dh *libusb.DeviceHandle) (int, error) {
	log.Printf("Resetting device\n")
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	ret, err := dh.ControlTransfer(
		requestType, byte(commandReset), 0x0, 0x0, []byte{0x00}, 1, timeout)
	if err != nil {
		return ret, fmt.Errorf("Error resetting devices %s", err)
	}
	return ret, nil
}

// Status retrieves the status of the device and clears the error
// indicators.
func Status(dh *libusb.DeviceHandle) (byte, error) {
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	data := make([]byte, 2)
	dh.ControlTransfer(
		requestType, byte(commandGetStatus), 0x0, 0x0, data, len(data), timeout)
	status := binary.LittleEndian.Uint16(data)
	return byte(status), nil
}

// SerialNumber retrieves the serial number via a control transfer using the
// serial command (0x48) as opposed to using the libusb serial number.
func SerialNumber(dh *libusb.DeviceHandle) (string, error) {
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	data := make([]byte, 8)
	dh.ControlTransfer(
		requestType, byte(commandSerialNum), 0x0, 0x0, data, len(data), timeout)
	return string(data), nil
}

// UpgradeFirmware places the device inthe firmate upgrade mode by erasing a
// portion of the program memory. The next time the device is reset, it will
// enumerate in the bootloader and is unusable as a DAQ device until new
// firmware is loaded.
func UpgradeFirmware(dh *libusb.DeviceHandle) error {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	key := uint16(0xadad)
	_, err := dh.ControlTransfer(
		requestType, byte(commandUpgradeFirmware), key, 0x0, []byte{}, 0, timeout)
	if err != nil {
		return fmt.Errorf("Error enabling upgrade firmware mode %s", err)
	}
	return nil
}
