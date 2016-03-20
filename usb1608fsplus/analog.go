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
	maxFrequency     = 500000
	defaultFrequency = 5000
)

type channel struct {
	Enabled     bool
	Range       voltageRange
	Description string
}

type channels [8]channel

type analogInput struct {
	DAQer
	Frequency         float64
	TransferMode      TransferMode
	Trigger           TriggerType
	UseExternalPacer  bool
	OutputPacerOnSync bool
	DebugMode         bool
	Stall             Stall
	Channels          channels
}

func (daq *usb1608fsplus) NewAnalogInput(freq float64) *analogInput {
	var channels [8]channel
	for i := 0; i < len(channels); i++ {
		channels[i].Range = Range10V
	}
	analogInput := analogInput{
		DAQer:             daq,
		Frequency:         freq,
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

type Stall byte

const (
	StallOnOverrun Stall = 0x0
	StallInhibited Stall = 0x1
)

func (ai *analogInput) EnabledChannels() byte {
	return ai.Channels.Enabled()
}

func (channels *channels) Enabled() byte {
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
func (ai *analogInput) Options() byte {
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
func (ai *analogInput) StartScan(numScans int) error {
	freq := ai.Frequency
	if ai.UseExternalPacer {
		freq = 0
	}
	data := packScanData(numScans, freq, ai.EnabledChannels(), ai.Options())
	log.Printf("packScanData = % x", data)
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

func (ai *analogInput) NumEnabledChannels() int {
	numEnabledChannels := 0
	for _, channel := range ai.Channels {
		if channel.Enabled {
			numEnabledChannels++
		}
	}
	return numEnabledChannels
}

// ReadScan reads the analog input data for the given number of scans
func (ai *analogInput) ReadScan(numScans int) ([]byte, error) {
	bytesInWord := 2
	log.Printf("There are %d channels enabled.", ai.NumEnabledChannels())
	wordsToRead := numScans * ai.NumEnabledChannels()
	bytesToRead := wordsToRead * bytesInWord
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
		log.Printf("Block transfer is expecting %d bytes", len(data))
		bytesReceived, err := ai.Read(data)
		log.Printf("Block transfer received %d bytes", bytesReceived)
		if err != nil {
			return data, fmt.Errorf("Problem with bulk scan %s", err)
		}
		if bytesReceived != bytesToRead {
			log.Printf("Excpected %d bytes but received %d bytes", len(data), bytesReceived)
			log.Printf("Last few bytes = %d %d %d %d", data[len(data)-4], data[len(data)-3], data[len(data)-2], data[len(data)-1])
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
	log.Printf("Bulk transfer is multiple of wMaxPacketSize %d", maxBulkTransferPacketSize)
	if (bytesToRead%maxBulkTransferPacketSize) == 0 && (status&byte(scanRunning) == 0) {
		log.Printf("Scan is not running so read a few bytes")
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
func (ai *analogInput) Close() error {
	return ai.StopScan()
}

// StopAnalogScan stops the analog input scan if running.
func (ai *analogInput) StopScan() error {
	_, err := ai.SendCommandToDevice(commandAnalogStopScan, nil)
	if err != nil {
		return fmt.Errorf("Error stopping analog input scan %s", err)
	}
	return nil
}

// ClearScanBuffer clears the internal scan endpoint FIFO buffer
func (ai *analogInput) ClearScanBuffer() error {
	_, err := ai.SendCommandToDevice(commandAnalogClearBuffer, nil)
	if err != nil {
		return fmt.Errorf("Error clearing analog input scan FIFO buffer %s", err)
	}
	return nil
}

// SetScanRanges writes the scan ranges to the USB-1608FS-Plus
func (ai *analogInput) SetScanRanges() error {
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

func (ai *analogInput) ScanRanges() ([]byte, error) {
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
func (daq *usb1608fsplus) ReadAnalogInput(channel int, rng voltageRange) (uint, error) {
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

func (ai *analogInput) ConfigureChannel(ch int, enabled bool, inputRange int, description string) error {

	// Verify valid channel number
	if ch < 0 || ch >= len(ai.Channels) {
		return fmt.Errorf("Channel %d outside valid range", ch)
	}

	// Verify valid input voltage range
	availableRanges := map[int]voltageRange{
		10: Range10V,
		5:  Range5V,
		2:  Range2V,
		1:  Range1V,
	}
	inputVoltageRange, ok := availableRanges[inputRange]
	if !ok {
		return fmt.Errorf("Voltage input range %d is invalid.", inputRange)
	}

	// Configure the channel
	ai.Channels[ch].Enabled = enabled
	ai.Channels[ch].Range = inputVoltageRange
	ai.Channels[ch].Description = description

	return nil
}
