// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/gotmc/libusb"
)

// BlinkLED blinks the LED the given number of times. Note, the LED starts
// being unlit, but will end being lit.
func (daq *USB1608fsplus) BlinkLED(blinks int) (int, error) {
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
func (daq *USB1608fsplus) Status() (byte, error) {
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	data := make([]byte, 2)
	daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandGetStatus), 0x0, 0x0, data, len(data), daq.Timeout)
	status := DecodeWord(data)
	return byte(status), nil
}

// SerialNumber retrieves the serial number via a control transfer using the
// serial command (0x48) as opposed to using the libusb serial number.
func (daq *USB1608fsplus) SerialNumber() (string, error) {
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
func (daq *USB1608fsplus) UpgradeFirmware() error {
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

// VoltsFromWord takes a 2-byte binary sequence and converts it to a voltage
// taking into account the offset, slope, and voltage range.
func VoltsFromWord(data []byte, voltageRange VoltageRange,
	slope, offset float64) (float64, error) {
	if len(data) != bytesPerWord {
		return 0.0, fmt.Errorf("binary value must be %d bytes", bytesPerWord)
	}
	rawValue := DecodeWord(data)
	adjustedValue := adjustRawValue(rawValue, slope, offset)
	return Volts(adjustedValue, voltageRange), nil
}

// RawVoltsFromWord converts the 2 byte binary value into the voltage for the
// given range
func RawVoltsFromWord(data []byte, voltageRange VoltageRange) (float64, error) {
	if len(data) != bytesPerWord {
		return 0.0, fmt.Errorf("binary value must be %d bytes", bytesPerWord)
	}
	rawValue := DecodeWord(data)
	return Volts(int(rawValue), voltageRange), nil
}

// Volts converts the unsigned binary value into the voltage for the given
// range. The given uint16 may or may not be a voltage that has been adjusted
// for the DAQ's gain and offset.
func Volts(b int, voltageRange VoltageRange) float64 {
	signedInt := b - converter
	value := VoltageMultiplier[voltageRange] * float64(signedInt) / converter
	return value
}

// DecodeWord decodes a 2-byte word into its uint16 equivalent. The analog
// inputs for the USB-1608FS-Plus have 16-bit resolution, so this is used in
// the conversion of DAQ analog readings into raw voltage values. However, the
// voltage value will still need to be adjusted for the gain and offset found
// in the gain table of the DAQ.
func DecodeWord(data []byte) uint16 {
	return binary.LittleEndian.Uint16(data)
}

// EncodeWord encodes an analog voltage value (a uint16) into a 2-byte
// sequence.
func EncodeWord(i uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, i)
	return b
}

// adjustRawValue takes a raw value as an int and adjusts it to account for the
// gain and offset.
func adjustRawValue(value uint16, slope, offset float64) int {
	adjFloat := float64(value)*slope + offset
	return roundFloatToInt(adjFloat)
}

func roundFloatToInt(f float64) int {
	if math.Abs(f) < 0.5 {
		return 0
	}
	return int(f + math.Copysign(0.5, f))
}
