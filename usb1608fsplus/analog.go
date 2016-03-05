// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"

	"github.com/gotmc/libusb"
)

const (
	maxFrequency = 500000
)

// ReadAnalogInput reads the value of an analog input channel. This command
// will result in a bus stall if an AInScan is currenty running.
func ReadAnalogInput(dh *libusb.DeviceHandle, channel int, rng voltageRange) (uint, error) {
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	data := make([]byte, 2)
	_, err := dh.ControlTransfer(
		requestType, byte(commandAnalogInput), uint16(channel), uint16(rng), data, len(data), timeout)
	if err != nil {
		return 0, fmt.Errorf("Error reading analog input %s", err)
	}
	value := binary.LittleEndian.Uint16(data)
	return uint(value), nil
}

// StartAnalogScan starts an analog input scan. If an AInScan is currently
// running, the bus will stall. The USB-1608FS-Plus will not generate an
// internal pacer faster than 100 kHz.
//
// The ADC is paced such that the pacer controls the ADC conversions.  The
// internal pacer rate is set by an internal 32-bit timer running at a
// base rate of 40 MHz.  The timer is controlled by pacer_period.  This
// value is the period of the scan and the ADCs are clocked at this rate.
// A pulse will be output at
// the SYNC pin at every pacer_period interval if SYNC is configred as an
// output.  The equation for calucating pacer_period is:
//
// pacer_period = [40 MHz / (A/D frequency)] - 1
//
/*
   If pacer_period is set to 0, the device does not generate an A/D
   clock.  It uses the SYNC pin as an input and the user must
   provide the pacer source.  The A/Ds acquire data on every rising
   edge of SYNC; the maximum allowable input frequency is 100 kHz.

   The data will be returned in packets untilizing a bulk endpoint.
   The data will be in the format:

   lowchannel sample 0: lowchannel + 1 sample 0: ... : hichannel sample 0
   lowchannel sample 1: lowchannel + 1 sample 1: ... : hichannel sample 1
   ...
   lowchannel sample n: lowchannel + 1 sample n: ... : hichannel sample n

   The scan will not begin until the AInScan Start command is sent
   and any trigger conditions are met.  Data will be sent until
   reaching the specified count or an usbAInScanStop_USB1608FS_Plus()
   command is sent.

   The external trigger may be used to start the scan.  If enabled,
   the device will wait until the appropriate trigger condition is
   detected than begin sampling data at the specified rate.  No
   packets will be sent until the trigger is detected.

   In block transfer mode, the data is sent in 64-byte packets as
   soon as data is available from the A/D.  In immediate transfer
   mode, the data is sent after each scan, resulting in packets that
   are 1-8 samples (2-16 bytes) long.  This mode should only be used
   for low pacer rates, typically under 100 Hz, because overruns
   will occur if the rate is too high.

   There is a 32,768 sample FIFO, and scans under 32 kS can be
   performed at up to 100 kHz*8 channels without overrun.

   Overruns are indicated by the device stalling the bulk endpoint
   during the scan.  The host may read the status to verify and ust
   clear the stall condition before further scan can be performed.
*/
func StartAnalogScan(
	dh *libusb.DeviceHandle, numScans int, frequency float64, channels byte, options byte,
) error {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	data := packScanData(numScans, frequency, channels, options)
	if len(data) != 10 {
		fmt.Errorf("StartAnalogScan data is not 10 bytes long.")
	}
	err := StopAnalogScan(dh)
	if err != nil {
		return fmt.Errorf("Error stopping analog scan prior to starting a new scan %s", err)
	}
	err = ClearScanBuffer(dh)
	if err != nil {
		return fmt.Errorf("Error clearing buffer prior to starting a new scan %s", err)
	}
	_, err = dh.ControlTransfer(
		requestType, byte(commandAnalogStartScan), 0x0, 0x0, data, len(data), timeout)
	if err != nil {
		return fmt.Errorf("Error starting analog input scan %s", err)
	}
	return nil
}

// ReadScan reads the data from an analog scan
func ReadScan(
	dh *libusb.DeviceHandle, ep *libusb.EndpointDescriptor, numScans int, numChannels int, options byte,
) ([]byte, error) {
	bytesInWord := 2
	wordsToRead := numScans * numChannels
	bytesToRead := wordsToRead * bytesInWord
	var data = make([]byte, bytesToRead)

	if options&byte(scanImmediateTransferMode) > 0 {
		// Immediate transfer mode scan
		for i := 0; i < wordsToRead; i++ {
			var word = make([]byte, bytesInWord)
			bytesReceived, err := dh.BulkTransfer(
				ep.EndpointAddress,
				word,
				bytesInWord,
				timeout,
			)
			if err != nil {
				return data, fmt.Errorf("Problem with immediate scan %s", err)
			}
			if bytesReceived != bytesInWord {
				return data, fmt.Errorf("Didn't transfer 2 bytes %s", err)
			}
			data[i] = word[0]
			data[i+1] = word[1]
		}
	} else {
		bytesReceived, err := dh.BulkTransfer(
			ep.EndpointAddress,
			data,
			bytesToRead,
			timeout,
		)
		if err != nil {
			return data, fmt.Errorf("Problem with bulk scan %s", err)
		}
		if bytesReceived != bytesToRead {
			return data, fmt.Errorf("Didn't transfer %d bytes %s", bytesToRead, err)
		}
	}
	status, err := Status(dh)
	if err != nil {
		fmt.Errorf("Error getting status during analog bulk read %s", err)
	}
	// If bytesToRead is a multiple of wMaxPacketSize the device will send a zero
	// byte packet.
	if (bytesToRead%maxBulkTransferPacketSize) == 0 && (status&byte(scanRunning) == 0) {
		_, _, _ = dh.BulkTransferIn(
			ep.EndpointAddress,
			bytesInWord,
			100,
		)
	}
	if status&byte(scanOverrun) != 0 {
		log.Printf("Analog AIn scan overrun.\n")
		StopAnalogScan(dh)
		ClearScanBuffer(dh)
	}

	return data, nil
}

// StopAnalogScan stops the analog input scan if running.
func StopAnalogScan(dh *libusb.DeviceHandle) error {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	_, err := dh.ControlTransfer(
		requestType, byte(commandAnalogStopScan), 0x0, 0x0, []byte{0}, 0, timeout)
	if err != nil {
		return fmt.Errorf("Error stopping analog input scan %s", err)
	}
	return nil
}

// ClearScanBuffer clears the internal scan endpoint FIFO buffer
func ClearScanBuffer(dh *libusb.DeviceHandle) error {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	_, err := dh.ControlTransfer(
		requestType, byte(commandAnalogClearBuffer), 0x0, 0x0, []byte{0}, 0, timeout)
	if err != nil {
		return fmt.Errorf("Error clearing analog input scan FIFO buffer %s", err)
	}
	return nil
}

// ConfigAnalogScan read or writes the analog input configuration. This command
// will result in a bus stall if an AIn scan is currently running.
func ConfigAnalogScan(dh *libusb.DeviceHandle, ranges []byte) error {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice, libusb.Vendor, libusb.DeviceRecipient)
	_, err := dh.ControlTransfer(
		requestType, byte(commandAnalogConfig), 0x0, 0x0, ranges, 8, timeout)
	if err != nil {
		return fmt.Errorf("Error writing Ain config %s", err)
	}
	return nil
}

func ReadScanRanges(dh *libusb.DeviceHandle) ([]byte, error) {
	var ranges = make([]byte, 8)
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	_, err := dh.ControlTransfer(
		requestType, byte(commandAnalogConfig), 0x0, 0x0, ranges, 8, timeout)
	if err != nil {
		return ranges, fmt.Errorf("Error reading Ain config %s", err)
	}
	return ranges, nil
}

func packScanData(numScans int, frequency float64, channels byte, options byte) []byte {
	// FIXME(mdr): I should probably use binary.Write() instead of using the
	// brute force method. <https://golang.org/pkg/encoding/binary/#example_Write_multi>

	// Convert numScans from int to []byte
	binaryNumScans := make([]byte, 4)
	binary.LittleEndian.PutUint32(binaryNumScans, uint32(numScans))

	pacerPeriod := calculatePacerPeriod(frequency)
	binaryPacerPeriod := make([]byte, 4)
	binary.LittleEndian.PutUint32(binaryPacerPeriod, uint32(pacerPeriod))

	return []byte{
		binaryNumScans[0],
		binaryNumScans[1],
		binaryNumScans[2],
		binaryNumScans[3],
		binaryPacerPeriod[0],
		binaryPacerPeriod[1],
		binaryPacerPeriod[2],
		binaryPacerPeriod[3],
		channels,
		options,
	}
}

func calculatePacerPeriod(frequency float64) int {
	if frequency > maxFrequency {
		frequency = maxFrequency
	}
	if frequency > 0 {
		return round((40e6 / float32(frequency)) - 1)
	}
	return 0
}

func round(f float32) int {
	fAsFloat64 := float64(f)
	if math.Abs(fAsFloat64) < 0.5 {
		return 0
	}
	return int(fAsFloat64 + math.Copysign(0.5, fAsFloat64))
}
