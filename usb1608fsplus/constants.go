// Copyright (c) 2016-2017 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

import (
	"encoding/json"
	"fmt"
)

// Since each binary encoded value is 16-bits (2 bytes), the converter value is
// 0x8000, which is 32768.
const (
	maxFrequency                   = 500000
	defaultFrequency               = 10000
	bytesPerWord                   = 2
	converter                      = 32768
	numChannels                    = 8
	lastChannel               byte = 0x80
	maxBulkTransferPacketSize      = 64
)

const (
	maxNumADChannels = 8  // max number of A/D channels in device
	maxNumGainLevels = 8  // max number of gain levels in device
	maxPacketSize    = 64 // max packet size for FS device
)

// InternalPacer models whether the internal pacer is on or off.
type InternalPacer byte

// Available settings for the InternalPacer
const (
	InternalPacerOff InternalPacer = 0x0
	InternalPacerOn  InternalPacer = 0x1
)

// Stall instructs the MCC DAQ on what to do when it encounters a stall.
type Stall byte

// Available options for a stall
const (
	StallOnOverrun Stall = 0x0
	StallInhibited Stall = 0x1
)

// UnmarshalJSON implements the Unmarshaler interface for Stall.
func (st *Stall) UnmarshalJSON(data []byte) error {
	// Extract the boolean from data.
	var stall bool
	if err := json.Unmarshal(data, &stall); err != nil {
		return fmt.Errorf("stall should be a boolean, got %s", data)
	}

	st.OnOverrun(stall)
	return nil
}

// OnOverrun enables StallOnOverrun using a boolean.
func (st *Stall) OnOverrun(t bool) {
	if t {
		*st = StallOnOverrun
	} else {
		*st = StallInhibited
	}
}

// MarshalJSON implements the Marshaler interface for Stall.
func (st *Stall) MarshalJSON() ([]byte, error) {
	stall := false
	if *st == StallOnOverrun {
		stall = true
	}
	return json.Marshal(stall)
}

// TransferMode declares whether to perform a block or immediate transfer of
// data from the MCC DAQ.
type TransferMode byte

// Available transfer modes.
const (
	BlockTransfer     TransferMode = 0x0
	ImmediateTransfer TransferMode = 0x1
)

// UnmarshalJSON implements the Unmarshaler interface for TransferMode by
// converting the JSON boolean value into the correct TransferMode value.
func (mode *TransferMode) UnmarshalJSON(data []byte) error {
	// Extract the boolean from data.
	var block bool
	if err := json.Unmarshal(data, &block); err != nil {
		return fmt.Errorf("block_transfer should be a boolean, got %s", data)
	}
	mode.BlockMode(block)
	return nil
}

// MarshalJSON implements the Marshaler interface for TransferMode.
func (mode *TransferMode) MarshalJSON() ([]byte, error) {
	isBlockTransfer := false
	if *mode == BlockTransfer {
		isBlockTransfer = true
	}
	return json.Marshal(isBlockTransfer)
}

// BlockMode uses a boolean to either set block or immediate transfer mode.
func (mode *TransferMode) BlockMode(block bool) {
	if block {
		*mode = BlockTransfer
	} else {
		*mode = ImmediateTransfer
	}
}

// TriggerType identifies the type of trigger to use.
type TriggerType byte

// Available TriggerType options.
const (
	NoExternalTrigger  TriggerType = 0x0
	RisingEdgeTrigger  TriggerType = 0x1
	FallingEdgeTrigger TriggerType = 0x2
	HighLevelTrigger   TriggerType = 0x3
	LowLevelTrigger    TriggerType = 0x4
)

// TriggerTypes maps a string to the actual TriggerType.
var TriggerTypes = map[string]TriggerType{
	"none":    NoExternalTrigger,
	"rising":  RisingEdgeTrigger,
	"falling": FallingEdgeTrigger,
	"high":    HighLevelTrigger,
	"low":     LowLevelTrigger,
}

// TriggerTypeStrings maps a TriggerType to a string representation for use by
// Stringer.
var TriggerTypeStrings = map[TriggerType]string{
	NoExternalTrigger:  "none",
	RisingEdgeTrigger:  "rising",
	FallingEdgeTrigger: "falling",
	HighLevelTrigger:   "high",
	LowLevelTrigger:    "low",
}

// String implements the Stringer interface for TriggerType.
func (t TriggerType) String() string {
	return TriggerTypeStrings[t]
}

// UnmarshalJSON implements the Unmarshaler interface for TriggerType by taking
// a string that matches a key in the TriggerTypes map and finding the
// appropriate TriggerType value.
func (t *TriggerType) UnmarshalJSON(data []byte) error {
	// Extract the string from data.
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("range should be a string, got %s", data)
	}
	return t.SetType(s)
}

// SetType sets the type of trigger using a string.
func (t *TriggerType) SetType(s string) error {
	got, ok := TriggerTypes[s]
	if !ok {
		return fmt.Errorf("invalid string `%q` for TriggerType", s)
	}
	*t = got
	return nil
}

// MarshalJSON implements the Marshaler interface for TriggerType.
func (t *TriggerType) MarshalJSON() ([]byte, error) {
	return json.Marshal(TriggerTypeStrings[*t])
}

type analogInputSetup byte

// Analog input setup
const (
	singleEnded  analogInputSetup = 0
	differential analogInputSetup = 1
	calibration  analogInputSetup = 3
)

// VoltageRange is a byte value used by the DAQ to determine the voltage range
// for the analog input.
type VoltageRange byte

// Available voltage ranges
const (
	Range10V     VoltageRange = 0x0 // ±10V
	Range5V      VoltageRange = 0x1 // ±5V
	Range2_5V    VoltageRange = 0x2 // ±2.5V
	Range2V      VoltageRange = 0x3 // ±2V
	Range1_25V   VoltageRange = 0x4 // ±1.25V
	Range1V      VoltageRange = 0x5 // ±1V
	Range0_625V  VoltageRange = 0x6 // ±0.625V
	Range0_3125V VoltageRange = 0x7 // ±0.3125V
)

// InputRanges maps the string keys that can be used in a JSON config file to
// the VoltageRange byte values.
var InputRanges = map[string]VoltageRange{
	"10V": Range10V,
	"5V":  Range5V,
	"2V":  Range2V,
	"1V":  Range1V,
}

var voltageRangeJSON = map[VoltageRange]string{
	Range10V: "10V",
	Range5V:  "5V",
	Range2V:  "2V",
	Range1V:  "1V",
}

var voltageRanges = map[VoltageRange]string{
	Range10V: "±10V",
	Range5V:  "±5V",
	Range2V:  "±2V",
	Range1V:  "±1V",
}

// String implements the Stringer interface for VoltageRange
func (v VoltageRange) String() string {
	return voltageRanges[v]
}

// VoltageMultiplier maps a VoltageRange to the float64 multipler value for
// that range.
var VoltageMultiplier = map[VoltageRange]float64{
	Range10V: 10.0,
	Range5V:  5.0,
	Range2V:  2.0,
	Range1V:  1.0,
}

type statusBit byte

// Status bit values
const (
	scanRunning statusBit = 0x1 << 1
	scanOverrun statusBit = 0x1 << 2
)
