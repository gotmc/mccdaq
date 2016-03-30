// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"

	"github.com/gotmc/libusb"
)

const (
	maxFrequency     = 500000
	defaultFrequency = 10000
)

type Channel struct {
	Enabled     bool         `json:"enabled"`
	Range       VoltageRange `json:"range"`
	Description string       `json:"desc"`
}

type Channels [8]Channel

type AnalogInput struct {
	DAQer             `json:"-"`
	Frequency         float64      `json:"freq"`
	TransferMode      TransferMode `json:"block_transfer"`
	Trigger           TriggerType  `json:"trigger"`
	UseExternalPacer  bool         `json:"ext_pacer"`
	OutputPacerOnSync bool         `json:"output_sync"`
	DebugMode         bool         `json:"debug_mode"`
	Stall             Stall        `json:"stall_overrun"`
	Channels          Channels     `json:"channels"`
}

func (st *Stall) UnmarshalJSON(data []byte) error {
	// Extract the boolean from data.
	var stall bool
	if err := json.Unmarshal(data, &stall); err != nil {
		return fmt.Errorf("stall should be a boolean, got %s", data)
	}

	if stall {
		*st = StallOnOverrun
	} else {
		*st = StallInhibited
	}
	return nil
}

func (mode *TransferMode) UnmarshalJSON(data []byte) error {
	// Extract the boolean from data.
	var block bool
	if err := json.Unmarshal(data, &block); err != nil {
		return fmt.Errorf("block_transfer should be a boolean, got %s", data)
	}

	if block {
		*mode = BlockTransfer
	} else {
		*mode = ImmediateTransfer
	}
	return nil
}

// UnmarshalJSON implements the Unmarshaler interface for VoltageRange by
// taking a string that matches a key in the InputRanges map and finding the
// appropriate VoltageRange value.
func (vr *VoltageRange) UnmarshalJSON(data []byte) error {
	// Extract the string from data.
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("range should be a string, got %s", data)
	}

	// Ensure the provided string matches one of the keys in the map
	got, ok := InputRanges[s]
	if !ok {
		return fmt.Errorf("Invalid VoltageRange %q", s)
	}
	// Set the voltage range to the value found in the map per the key
	*vr = got
	return nil
}

// UnmarshalJSON implements the Unmarshaler interface for TriggerType by taking
// a string that matches a key in the TriggerTypes map and finding the
// appropriate TriggerType value.
func (trigger *TriggerType) UnmarshalJSON(data []byte) error {
	// Extract the string from data.
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("range should be a string, got %s", data)
	}

	got, ok := TriggerTypes[s]
	if !ok {
		return fmt.Errorf("Invalid TriggerType %q", s)
	}
	*trigger = got
	return nil
}

// NewAnalogInput is used to create a new AnalogInput for the given DAQer.
func (daq *usb1608fsplus) NewAnalogInput() *AnalogInput {
	var channels [8]Channel
	for i := 0; i < len(channels); i++ {
		channels[i].Range = Range10V
	}
	analogInput := AnalogInput{
		DAQer:             daq,
		Frequency:         defaultFrequency,
		TransferMode:      BlockTransfer,
		Trigger:           NoExternalTrigger,
		UseExternalPacer:  false,
		OutputPacerOnSync: false,
		DebugMode:         false,
		Stall:             StallOnOverrun,
		Channels:          channels,
	}
	return &analogInput
}

type TransferMode byte

const (
	BlockTransfer     TransferMode = 0x0
	ImmediateTransfer TransferMode = 0x1
)

type InternalPacer byte

const (
	InternalPacerOff InternalPacer = 0x0
	InternalPacerOn  InternalPacer = 0x1
)

type TriggerType byte

const (
	NoExternalTrigger  TriggerType = 0x0
	RisingEdgeTrigger  TriggerType = 0x1
	FallingEdgeTrigger TriggerType = 0x2
	HighLevelTrigger   TriggerType = 0x3
	LowLevelTrigger    TriggerType = 0x4
)

var TriggerTypes = map[string]TriggerType{
	"none":    NoExternalTrigger,
	"rising":  RisingEdgeTrigger,
	"falling": FallingEdgeTrigger,
	"high":    HighLevelTrigger,
	"low":     LowLevelTrigger,
}

type Stall byte

const (
	StallOnOverrun Stall = 0x0
	StallInhibited Stall = 0x1
)

func (ai *AnalogInput) EnabledChannels() byte {
	return ai.Channels.Enabled()
}

func (channels *Channels) Enabled() byte {
	var enabledChannels byte
	for i, channel := range channels {
		if channel.Enabled {
			enabledChannels = enabledChannels | 0x1<<uint(i)
		}
	}
	return enabledChannels
}

// Options returns the analog input scan options byte containing the following
// bit fields:
//
//   Bit 0: Transfer mode (0 = block / 1 = immediate)
//   Bit 1: Pacer output to Sync pin (0 = off / 1 = on) ignored when using an
//   	      external clock for pacing
//   Bits 2-4: Trigger settings:
//               0: No trigger
//               1: Trigger on rising edge
//               2: Trigger on falling edge
//               3: Trigger on high level
//               4: Trigger on low level
//   Bit 5: Debug mode:
//            0 = off; output A/D data
//            1 = on; output incrementing counter
//   Bit 7: Stall on bulk endpoint overrun (0 = no / 1 = yes)
func (ai *AnalogInput) Options() byte {
	transferMode := byte(ai.TransferMode)
	pacer := byte(InternalPacerOff)
	if ai.OutputPacerOnSync {
		pacer = byte(InternalPacerOn)
	}
	trigger := byte(ai.Trigger)
	debug := byte(0x0)
	if ai.DebugMode {
		debug = byte(0x1)
	}
	stall := byte(ai.Stall)
	return transferMode<<0 | pacer<<1 | trigger<<2 | debug<<5 | stall<<7
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
	 If pacer_period is set to 0, the device does not generate an A/D clock.  It
	 uses the SYNC pin as an input and the user must provide the pacer source.
	 The A/Ds acquire data on every rising edge of SYNC; the maximum allowable
	 input frequency is 100 kHz.

	 The data will be returned in packets untilizing a bulk endpoint.  The data
	 will be in the format:

   lowchannel sample 0: lowchannel + 1 sample 0: ... : hichannel sample 0
   lowchannel sample 1: lowchannel + 1 sample 1: ... : hichannel sample 1
   ...
   lowchannel sample n: lowchannel + 1 sample n: ... : hichannel sample n

	 The scan will not begin until the AInScan Start command is sent and any
	 trigger conditions are met.  Data will be sent until reaching the specified
	 count or an usbAInScanStop_USB1608FS_Plus() command is sent.

	 The external trigger may be used to start the scan.  If enabled, the device
	 will wait until the appropriate trigger condition is detected then begin
	 sampling data at the specified rate.  No packets will be sent until the
	 trigger is detected.

	 In block transfer mode, the data is sent in 64-byte packets as soon as data
	 is available from the A/D.  In immediate transfer mode, the data is sent
	 after each scan, resulting in packets that are 1-8 samples (2-16 bytes)
	 long.  This mode should only be used for low pacer rates, typically under
	 100 Hz, because overruns will occur if the rate is too high.

	 There is a 32,768 sample FIFO, and scans under 32 kS can be performed at up
	 to 100 kHz*8 channels without overrun.

	 Overruns are indicated by the device stalling the bulk endpoint during the
	 scan.  The host may read the status to verify and clear the stall condition
   before further scan can be performed.
*/
func (ai *AnalogInput) StartScan(numScans int) error {
	freq := ai.Frequency
	if ai.UseExternalPacer {
		freq = 0
	}
	data := packScanData(numScans, freq, ai.EnabledChannels(), ai.Options())
	if len(data) != 10 {
		fmt.Errorf("StartAnalogScan data is not 10 bytes long.")
	}
	err := ai.StopScan()
	if err != nil {
		return fmt.Errorf("Error stopping analog scan prior to starting a new scan %s", err)
	}
	err = ai.ClearScanBuffer()
	if err != nil {
		return fmt.Errorf("Error clearing buffer prior to starting a new scan %s", err)
	}
	_, err = ai.SendCommandToDevice(commandAnalogStartScan, data)
	if err != nil {
		return fmt.Errorf("Error starting analog input scan %s", err)
	}
	return nil
}

func (ai *AnalogInput) NumEnabledChannels() int {
	numEnabledChannels := 0
	for _, channel := range ai.Channels {
		if channel.Enabled {
			numEnabledChannels++
		}
	}
	return numEnabledChannels
}

// ReadScan reads the analog input data for the given number of scans
func (ai *AnalogInput) ReadScan(numScans int) ([]byte, error) {
	bytesInWord := 2
	wordsToRead := numScans * ai.NumEnabledChannels()
	bytesToRead := wordsToRead * bytesInWord
	if (bytesToRead % maxBulkTransferPacketSize) != 0 {
		return nil, fmt.Errorf("Bytes to read not a multiple of maxBulkTransferPacketSize")
	}
	var data = make([]byte, bytesToRead)
	if ai.TransferMode == ImmediateTransfer {
		for i := 0; i < wordsToRead; i++ {
			var word = make([]byte, bytesInWord)
			bytesReceived, err := ai.Read(word)
			if err != nil {
				return data, fmt.Errorf("Problem with immediate scan %s", err)
			}
			if bytesReceived != bytesInWord {
				return data, fmt.Errorf("Didn't transfer 2 bytes %s", err)
			}
			data[i] = word[0]
			data[i+1] = word[1]
		}
	} else if ai.TransferMode == BlockTransfer {
		bytesReceived, err := ai.Read(data)
		if err != nil {
			return data, fmt.Errorf("Problem with bulk scan %s", err)
		}
		if bytesReceived != bytesToRead {
			return data, fmt.Errorf("Didn't transfer %d bytes: %s", bytesToRead, err)
		}
	} else {
		return data, fmt.Errorf("Bad transfer mode")
	}
	status, err := ai.Status()
	if err != nil {
		fmt.Errorf("Error getting status during analog bulk read %s", err)
	}
	// If bytesToRead is a multiple of wMaxPacketSize the device will send a zero
	// byte packet.
	if (bytesToRead%maxBulkTransferPacketSize) == 0 && (status&byte(scanRunning) == 0) {
		var data = make([]byte, bytesInWord)
		_, _ = ai.Read(data)
	}
	if status&byte(scanOverrun) != 0 {
		log.Printf("Analog AIn scan overrun.\n")
		ai.StopScan()
		ai.ClearScanBuffer()
	}

	return data, err
}

// Close stops the analog input scan if running.
func (ai *AnalogInput) Close() error {
	return ai.StopScan()
}

// StopAnalogScan stops the USB-1608FS-Plus's analog input scan if running.
func (ai *AnalogInput) StopScan() error {
	_, err := ai.SendCommandToDevice(commandAnalogStopScan, nil)
	if err != nil {
		return fmt.Errorf("Error stopping analog input scan %s", err)
	}
	return nil
}

// ClearScanBuffer clears the internal scan endpoint FIFO buffer
func (ai *AnalogInput) ClearScanBuffer() error {
	_, err := ai.SendCommandToDevice(commandAnalogClearBuffer, nil)
	if err != nil {
		return fmt.Errorf("Error clearing analog input scan FIFO buffer %s", err)
	}
	return nil
}

// SetScanRanges writes the scan ranges to the USB-1608FS-Plus
func (ai *AnalogInput) SetScanRanges() error {
	ranges := make([]byte, 8)
	for i, channel := range ai.Channels {
		ranges[i] = byte(channel.Range)
	}
	if len(ranges) != 8 {
		return fmt.Errorf("length of ranges slice is not 8 bytes")
	}
	_, err := ai.SendCommandToDevice(commandAnalogConfig, ranges)
	if err != nil {
		return fmt.Errorf("Error writing Ain config %s", err)
	}
	return nil
}

// ScanRanges reads the scan ranges (i.e., input voltage ranges for the analog
// inputs) from the USB-1608FS-Plus.
func (ai *AnalogInput) ScanRanges() ([]byte, error) {
	const bytesInRange = 8
	var ranges = make([]byte, bytesInRange)
	bytesRead, err := ai.ReadCommandFromDevice(commandAnalogConfig, ranges)
	if err != nil {
		return ranges, fmt.Errorf("Error reading Ain config: %s", err)
	}
	if bytesRead != bytesInRange {
		return ranges, fmt.Errorf("Wrong number of ranges: %s", err)
	}
	return ranges, nil
}

// packScanData creates the 10 byte configuration information needed by
// StartScan.
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
		return round((40e6 / frequency) - 1)
	}
	return 0
}

func round(f float64) int {
	if math.Abs(f) < 0.5 {
		return 0
	}
	return int(f + math.Copysign(0.5, f))
}

// ReadAnalogInput reads the value of an analog input channel. This command
// will result in a bus stall if an AInScan is currenty running.
func (daq *usb1608fsplus) ReadAnalogInput(channel int, rng VoltageRange) (uint, error) {
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	data := make([]byte, 2)
	_, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandAnalogInput), uint16(channel), uint16(rng), data, len(data), daq.Timeout)
	if err != nil {
		return 0, fmt.Errorf("Error reading analog input %s", err)
	}
	value := binary.LittleEndian.Uint16(data)
	return uint(value), nil
}

// ConfigureEnabledChannel both enables and configures a channel. This is a
// convenience method for ConfigureChannel that enables the channel.
func (ai *AnalogInput) ConfigureEnableChannel(ch int, voltage, description string) error {
	return ai.ConfigureChannel(ch, true, voltage, description)
}

// EnableChannel enables the given channel without changing any other channel
// configuration items.
func (ai *AnalogInput) EnableChannel(ch int) {
	ai.Channels[ch].Enabled = true
}

func (ai *AnalogInput) Voltages(data []byte) ([]float64, error) {
	// Check that the data is the right size given the number of enabled channels
	return nil, nil
}

// DisableChannel disables the given channel without changing any other channel
// configuration items.
func (ai *AnalogInput) DisableChannel(ch int) {
	ai.Channels[ch].Enabled = false
}

// ConfigureChannel configures the given channel setting its input voltage
// range, description, and whether or not the channel is enabled.
func (ai *AnalogInput) ConfigureChannel(
	ch int, enabled bool, voltage string, description string,
) error {
	// Return error if the voltage range is invalid
	inputRange, ok := InputRanges[voltage]
	if !ok {
		return fmt.Errorf("Voltage input range `%s` is invalid.", inputRange)
	}
	return ai.configureChannel(ch, enabled, inputRange, description)
}

// configureChannel configures the given channel like ConfigureChannel but
// takes a VoltageRange instead of a string for the input voltage range.
func (ai *AnalogInput) configureChannel(
	ch int, enabled bool, voltage VoltageRange, description string,
) error {

	// Return error if the channel is invalid
	if ch < 0 || ch >= len(ai.Channels) {
		return fmt.Errorf("Channel %d outside valid range", ch)
	}

	// Configure the channel
	ai.Channels[ch].Enabled = enabled
	ai.Channels[ch].Range = voltage
	ai.Channels[ch].Description = description

	return nil
}
