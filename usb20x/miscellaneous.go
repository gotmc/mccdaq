// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb20x

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/gotmc/libusb"
)

func byteSlice(i int) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(i))
	return b
}

// BlinkLED blinks the LED the given number of times. Note, the LED starts
// being unlit, but will end being lit.
func (daq *usb20x) BlinkLED(blinks int) (int, error) {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	// data := byteSlice(blinks)
	data := make([]byte, 1)
	data[0] = byte(blinks)

	ret, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandBlinkLED), 0x0, 0x0, data, len(data), daq.Timeout)
	if err != nil {
		return ret, fmt.Errorf("Error blinking LED %s", err)
	}
	return ret, nil
}

// Status retrieves the status of the device and clears the error
// indicators.
func (daq *usb20x) Status() (byte, error) {
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	data := make([]byte, 2)
	daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandGetStatus), 0x0, 0x0, data, len(data), daq.Timeout)
	status := binary.LittleEndian.Uint16(data)
	return byte(status), nil
}

// SerialNumber retrieves the serial number via a control transfer using the
// serial command (0x48) as opposed to using the libusb serial number.
func (daq *usb20x) SerialNumber() (string, error) {
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	data := make([]byte, 8)
	daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandSerialNum), 0x0, 0x0, data, len(data), daq.Timeout)
	return string(data), nil
}

// UpgradeFirmware places the device inthe firmate upgrade mode by erasing a
// portion of the program memory. The next time the device is reset, it will
// enumerate in the bootloader and is unusable as a DAQ device until new
// firmware is loaded.
func (daq *usb20x) UpgradeFirmware() error {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	key := uint16(0xadad)
	_, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandUpgradeFirmware), key, 0x0, []byte{}, 0, daq.Timeout)
	if err != nil {
		return fmt.Errorf("Error enabling upgrade firmware mode %s", err)
	}
	return nil
}

// VoltsData converts the 2 byte binary value into the voltage for the given range
func VoltsData(data []byte, voltageRange VoltageRange) (float64, error) {
	if len(data) != 2 {
		return 0.0, fmt.Errorf("binary value must be 2 bytes")
	}
	b := int(binary.LittleEndian.Uint16(data))
	return Volts(b, voltageRange), nil
}

// Volts converts the unsigned binary value into the voltage for the given
// range
func Volts(b int, voltageRange VoltageRange) float64 {
	// Since each binary encoded value is 16-bits (2 bytes), the converter value
	// is 0x8000, which is 32768.
	const (
		converter = 32768
	)
	signedInt := b - converter
	value := VoltageMultiplier[voltageRange] * float64(signedInt) / converter
	return value
}

func float32From2Bytes(data []byte) float32 {
	return math.Float32frombits(uint32(binary.LittleEndian.Uint16(data)))
}
