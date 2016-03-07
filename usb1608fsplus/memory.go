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

type GainTable struct {
	Slope     [][]float64
	Intercept [][]float64
}

// BuildGainTable creates a multidimensional slice to store the slope
// and intercept for each range on each channel. The calibration coefficients
// are stored in onboard FLASH memory on the device in IEEE-754 4-byte floating
// point values.
func BuildGainTable(dh *libusb.DeviceHandle) (GainTable, error) {
	// TODO(mdr): Why are we reading only 4 bytes at a time in a loop? Why not
	// read all calibration memory at once and then decode the data as needed to
	// create the calibraiton gain table.
	var data []byte
	address := 0
	bytesPerValue := 4
	slope := make([][]float64, maxNumGainLevels)
	intercept := make([][]float64, maxNumGainLevels)
	for i := 0; i < maxNumGainLevels; i++ {
		slope[i] = make([]float64, maxNumADChannels)
		intercept[i] = make([]float64, maxNumADChannels)
		for j := 0; j < maxNumADChannels; j++ {
			data, _ = ReadCalMemory(dh, address, bytesPerValue)
			slope[i][j] = float64(convertBytesToFloat32(data))
			address += bytesPerValue
			data, _ = ReadCalMemory(dh, address, bytesPerValue)
			intercept[i][j] = float64(convertBytesToFloat32(data))
			address += bytesPerValue
		}
	}
	gainTable := GainTable{
		Slope:     slope,
		Intercept: intercept,
	}
	// In the c version, he reads from the device and sets the wMaxPacketSize,
	// which is a global variable
	// TODO(mdr): Do I need to do that as well?

	return gainTable, nil
}

// ReadCalMemory reads the nonvolatile calibration memory.
/*
   This command allows for reading and writing the nonvolatile
    calibration memory.  The cal memory is 768 bytes (address
    0-0x2FF).  The cal memory is write protected and must be unlocked
    in order to write the memory.  The unlock procedure is to write
    the unlock code 0xAA55 to address 0x300.  Writes to the entire
    memory range is then possible.  Write any other value to address
    0x300 to lock the memory after writing.
*/
func ReadCalMemory(dh *libusb.DeviceHandle, address int, count int) ([]byte, error) {
	data := make([]byte, count)
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	if count > 768 {
		return nil, fmt.Errorf("Max bytes is 768")
	}
	if address > 0x2ff {
		return nil, fmt.Errorf("Address must be in the range 0x0000 to 0x02FF")
	}
	dh.ControlTransfer(
		requestType, byte(commandCalibrationMemory), uint16(address), 0x0, data, count, timeout)
	return data, nil
}

func convertBytesToFloat32(data []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(data))
}
