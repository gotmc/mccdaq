// Copyright (c) 2016-2017 The mccdaq developers. All rights reserved.
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

// AnalogInput models an analog input for the MCC DAQ.
type AnalogInput struct {
	DAQ               DAQer        `json:"-"`
	Frequency         float64      `json:"freq"`
	TransferMode      TransferMode `json:"block_transfer"`
	Trigger           TriggerType  `json:"trigger"`
	UseExternalPacer  bool         `json:"ext_pacer"`
	OutputPacerOnSync bool         `json:"output_sync"`
	DebugMode         bool         `json:"debug_mode"`
	Stall             Stall        `json:"stall_overrun"`
	Channels          Channels     `json:"channels"`
}

// Channel models a single channel of an analog input.
type Channel struct {
	Enabled     bool         `json:"enabled"`
	Range       VoltageRange `json:"range"`
	Description string       `json:"desc"`
	Slopes      Slopes       `json:"slopes"`
	Intercepts  Intercepts   `json:"intercepts"`
}

// Intercepts contains the offsets based on the voltage range.
type Intercepts map[VoltageRange]float64

// Channels contains an array of each of the eight Channels.
type Channels [8]Channel

// Slopes contains the gains based on the voltage range.
type Slopes map[VoltageRange]float64

// MarshalJSON implements the Marshaler interface for Slopes.
func (s *Slopes) MarshalJSON() ([]byte, error) {
	sloper := make(map[string]float64)
	for k, v := range *s {
		sloper[voltageRangeJSON[k]] = v
	}
	return json.Marshal(sloper)
}

// MarshalJSON implements the Marshaler interface for Intercepts.
func (i *Intercepts) MarshalJSON() ([]byte, error) {
	intercepter := make(map[string]float64)
	for k, v := range *i {
		intercepter[voltageRangeJSON[k]] = v
	}
	return json.Marshal(intercepter)
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
	return vr.Set(s)
}

// Set sets the voltage range using a string.
func (vr *VoltageRange) Set(s string) error {
	// Ensure the provided string matches one of the keys in the map
	got, ok := InputRanges[s]
	if !ok {
		return fmt.Errorf("Invalid VoltageRange %q", s)
	}
	// Set the voltage range to the value found in the map per the key
	*vr = got
	return nil
}

// MarshalJSON implements the Marshaler interface for VoltageRange.
func (vr *VoltageRange) MarshalJSON() ([]byte, error) {
	return json.Marshal(voltageRangeJSON[*vr])
}

// NewAnalogInput is used to create a new AnalogInput for the given DAQer.
func (daq *USB1608fsplus) NewAnalogInput() (*AnalogInput, error) {
	gainTable, err := daq.BuildGainTable()
	if err != nil {
		return nil, fmt.Errorf("Error reading gain table from DAQ: %s", err)
	}
	var channels [numChannels]Channel
	for i := 0; i < len(channels); i++ {
		channels[i].Range = Range10V
		channels[i].Slopes = make(map[VoltageRange]float64)
		channels[i].Intercepts = make(map[VoltageRange]float64)
		// Loop through each range to get the slope and intercept for each channel
		for rng := 0; rng < len(gainTable.Slope); rng++ {
			channels[i].Slopes[VoltageRange(rng)] = gainTable.Slope[rng][i]
			channels[i].Intercepts[VoltageRange(rng)] = gainTable.Intercept[rng][i]
		}
	}
	analogInput := AnalogInput{
		DAQ:               daq,
		Frequency:         defaultFrequency,
		TransferMode:      BlockTransfer,
		Trigger:           NoExternalTrigger,
		UseExternalPacer:  false,
		OutputPacerOnSync: false,
		DebugMode:         false,
		Stall:             StallOnOverrun,
		Channels:          channels,
	}
	return &analogInput, nil
}

// EnabledChannels returns a byte as an 8-bit flag identifying the enabled
// analog input channels.
func (ai *AnalogInput) EnabledChannels() byte {
	return ai.Channels.Enabled()
}

// Enabled returns a byte as an 8-bit flag identifying the enabled analog input
// channels.
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
//   Bit 7: Stall on bulk endpoint overrun (0 = yes / 1 = no)
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

// StartScan starts an analog input scan. If an AInScan is currently
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
		return fmt.Errorf("scan data length is %d bytes; expected 10 bytes", len(data))
	}
	err := ai.StopScan()
	if err != nil {
		return fmt.Errorf("error stopping analog scan prior to starting a new scan: %s", err)
	}
	err = ai.ClearScanBuffer()
	if err != nil {
		return fmt.Errorf("error clearing buffer prior to starting a new scan %s", err)
	}
	_, err = ai.DAQ.SendCommandToDevice(commandAnalogStartScan, data)
	if err != nil {
		return fmt.Errorf("error starting analog input scan %s", err)
	}
	return nil
}

// NumEnabledChannels returns the number of enabled analog input channels on
// the DAQ.
func (ai *AnalogInput) NumEnabledChannels() int {
	numEnabledChannels := 0
	for _, channel := range ai.Channels {
		if channel.Enabled {
			numEnabledChannels++
		}
	}
	return numEnabledChannels
}

// Read reads the analog input data. The number of scans is based on the size
// of the given byte slice. This function replaces the old ReadScan(numScans int)
// ([]byte, error), since that put pressure on the garabage collector by requiring
// an allocation every time the function was called, since it returned a byte slice.
func (ai *AnalogInput) Read(p []byte) (n int, err error) {
	bytesToRead := len(p)
	wordsToRead := bytesToRead / bytesPerWord
	if (bytesToRead % maxBulkTransferPacketSize) != 0 {
		return n, fmt.Errorf("%d bytes to read is not a multiple of maxBulkTransferPacketSize",
			bytesToRead)
	}
	var data = make([]byte, bytesToRead)
	switch ai.TransferMode {
	case ImmediateTransfer:
		for i := 0; i < wordsToRead; i++ {
			var word = make([]byte, bytesPerWord)
			bytesReceived, err := ai.DAQ.Read(word)
			if err != nil {
				return n, fmt.Errorf("immediate scan error: %s", err)
			}
			n += bytesReceived
			if bytesReceived != bytesPerWord {
				return n, fmt.Errorf("immediate transfer of %d bytes instead of %d: %s",
					bytesReceived, bytesPerWord, err)
			}
			data[i] = word[0]
			data[i+1] = word[1]
		}
	case BlockTransfer:
		bytesReceived, err := ai.DAQ.Read(data)
		if err != nil {
			return n, fmt.Errorf("Problem with bulk scan %s", err)
		}
		n += bytesReceived
		if bytesReceived != bytesToRead {
			return n, fmt.Errorf("Didn't transfer %d bytes: %s", bytesToRead, err)
		}
	default:
		return n, fmt.Errorf("bad transfer mode")
	}
	status, err := ai.DAQ.Status()
	if err != nil {
		return n, fmt.Errorf("error getting status during analog bulk read %s", err)
	}
	// If bytesToRead is a multiple of wMaxPacketSize the device will send a zero
	// byte packet.
	if (bytesToRead%maxBulkTransferPacketSize) == 0 && (status&byte(scanRunning) == 0) {
		data := make([]byte, bytesPerWord)
		_, _ = ai.DAQ.Read(data)
	}
	if status&byte(scanOverrun) != 0 {
		log.Printf("Analog AIn scan overrun.\n")
		ai.StopScan()
		ai.ClearScanBuffer()
	}
	return n, err
}

// Close stops the analog input scan if running.
func (ai *AnalogInput) Close() error {
	return ai.StopScan()
}

// StopScan stops the USB-1608FS-Plus's analog input scan if running.
func (ai *AnalogInput) StopScan() error {
	_, err := ai.DAQ.SendCommandToDevice(commandAnalogStopScan, nil)
	if err != nil {
		return fmt.Errorf("Error stopping analog input scan %s", err)
	}
	return nil
}

// ClearScanBuffer clears the internal scan endpoint FIFO buffer
func (ai *AnalogInput) ClearScanBuffer() error {
	_, err := ai.DAQ.SendCommandToDevice(commandAnalogClearBuffer, nil)
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
	_, err := ai.DAQ.SendCommandToDevice(commandAnalogConfig, ranges)
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
	bytesRead, err := ai.DAQ.ReadCommandFromDevice(commandAnalogConfig, ranges)
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
func (daq *USB1608fsplus) ReadAnalogInput(channel int, rng VoltageRange) (uint, error) {
	requestType := libusb.BitmapRequestType(
		libusb.DeviceToHost, libusb.Vendor, libusb.DeviceRecipient)
	data := make([]byte, 2)
	_, err := daq.DeviceHandle.ControlTransfer(
		requestType, byte(commandAnalogInput), uint16(channel), uint16(rng), data, len(data), daq.Timeout)
	if err != nil {
		return 0, fmt.Errorf("Error reading analog input %s", err)
	}
	value := DecodeWord(data)
	return uint(value), nil
}

// ConfigureEnableChannel both enables and configures a channel. This is a
// convenience method for ConfigureChannel that enables the channel.
func (ai *AnalogInput) ConfigureEnableChannel(ch int, voltage, description string) error {
	return ai.ConfigureChannel(ch, true, voltage, description)
}

// EnableChannel enables the given channel without changing any other channel
// configuration items.
func (ai *AnalogInput) EnableChannel(ch int) {
	ai.Channels[ch].Enabled = true
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
		return fmt.Errorf("voltage input range `%s` is invalid", inputRange)
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
		return fmt.Errorf("channel %d outside valid range", ch)
	}

	// Configure the channel
	ai.Channels[ch].Enabled = enabled
	ai.Channels[ch].Range = voltage
	ai.Channels[ch].Description = description

	return nil
}

// RawVoltages converts the given binary data into a 2D slice of float64s. The
// binary data is two byte raw integer values by channel and then by scan. The
// 2D slice has dimensions of channel first and then number of scans.
func (ai *AnalogInput) RawVoltages(data []byte) ([][]float64, error) {
	// Check that the given data is a multiple of 2 bytes (1 word) by the number
	// of channels (8).
	if len(data)%(bytesPerWord*len(ai.Channels)) != 0 {
		return nil, fmt.Errorf("data len must be multiple of 2 bytes x 8 channels")
	}
	scans := len(data) / (bytesPerWord * len(ai.Channels))
	rawVoltages := make([][]float64, len(ai.Channels))
	for i := range ai.Channels {
		rawVoltages[i] = make([]float64, scans)
	}
	word := 0
	for scan := 0; scan < scans; scan++ {
		for i, ch := range ai.Channels {
			firstByte := word * bytesPerWord
			raw, err := RawVoltsFromWord(data[firstByte:firstByte+bytesPerWord], ch.Range)
			if err != nil {
				return rawVoltages, err
			}
			rawVoltages[i][scan] = raw
			word++
		}
	}
	return rawVoltages, nil
}

// Voltages calculates the actual voltage reading given the raw binary data,
// which is converted into a 2D slice by channel and scan taking into account
// the MCC DAQ's gain, offset, and range for each channel.
func (ai *AnalogInput) Voltages(data []byte) ([][]float64, error) {
	// Check that the given data is a multiple of 2 bytes (1 word) by the number
	// of channels (8).
	if len(data)%(bytesPerWord*len(ai.Channels)) != 0 {
		return nil, fmt.Errorf("data len must be multiple of 2 bytes x 8 channels")
	}
	scans := len(data) / (bytesPerWord * len(ai.Channels))
	voltages := make([][]float64, len(ai.Channels))
	for i := range ai.Channels {
		voltages[i] = make([]float64, scans)
	}
	word := 0
	slopes := make([]float64, len(ai.Channels))
	offsets := make([]float64, len(ai.Channels))
	for i, ch := range ai.Channels {
		slopes[i] = ch.Slopes[ch.Range]
		offsets[i] = ch.Intercepts[ch.Range]
	}
	for scan := 0; scan < scans; scan++ {
		for i, ch := range ai.Channels {
			firstByte := word * bytesPerWord
			voltage, err := VoltsFromWord(
				data[firstByte:firstByte+bytesPerWord], ch.Range, slopes[i], offsets[i])
			if err != nil {
				return voltages, err
			}
			voltages[i][scan] = voltage
			word++
		}
	}
	return voltages, nil
}

// Volts converts a two byte integer into a float64 accounting for the offset,
// slope, and range of the channel.
func (ch Channel) Volts(data []byte) (float64, error) {
	return VoltsFromWord(data, ch.Range, ch.Slopes[ch.Range], ch.Intercepts[ch.Range])
}
