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

// FIXME(mdr): Should I use float64 for the gain table? Since the gain table is
// stored using IEEE-754 4-byte floating point values, maybe we should use
// float32 here? Using float32 might just cause a problem when using this code
// on a RPi or other lower powered computing device.
type GainTable struct {
	Slope     [][]float64
	Intercept [][]float64
}

// BuildGainTable creates a multidimensional slice to store the slope
// and intercept for each range on each channel. The calibration coefficients
// are stored in onboard FLASH memory on the device in IEEE-754 4-byte floating
// point values.
func (daq *USB1608FSPlus) BuildGainTable() (GainTable, error) {
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
			data, _ = daq.ReadCalMemory(address, bytesPerValue)
			slope[i][j] = float64(convertBytesToFloat32(data))
			address += bytesPerValue
			data, _ = daq.ReadCalMemory(address, bytesPerValue)
			intercept[i][j] = float64(convertBytesToFloat32(data))
			address += bytesPerValue
		}
	}
	gainTable := GainTable{
		Slope:     slope,
		Intercept: intercept,
	}
	// The C version of the USB-1608FS-Plus driver reads from the device and sets
	// the wMaxPacketSize, which is a global variable.
	// TODO(mdr): Should I be doing that as well?

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
func (daq *USB1608FSPlus) ReadCalMemory(address int, count int) ([]byte, error) {
	data := make([]byte, count)
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient,
	)

	if !validCalMemoryRange(address, count) {
		return nil, fmt.Errorf(
			"Tyring to access outside calibration memory range 0x0000 to 0x02FF")
	}

	daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandCalibrationMemory), uint16(address), 0x0, data, count, timeout)
	return data, nil
}

func convertBytesToFloat32(data []byte) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(data))
}

func validCalMemoryRange(address, count int) bool {
	numCalMemoryBytes := 768
	maxCalMemoryLocation := 0x02ff // 768 bytes from 0x0000 to 0x02ff
	// Must read at least 1 byte and no more than 768 bytes
	if count <= 0 || count > numCalMemoryBytes {
		return false
	}
	if address < 0 || maxCalMemoryLocation < address+count-1 {
		return false
	}
	return true
}
