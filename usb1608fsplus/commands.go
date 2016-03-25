// Copyright (c) 2016 The mccdaq developers. All rights reserved.
// Project site: https://github.com/gotmc/mccdaq
// Use of this source code is governed by a MIT-style license that
// can be found in the LICENSE.txt file for the project.

package usb1608fsplus

type command byte

// Log level enumeration
const (
	// Digital I/O commands
	commandDigitalTristate command = 0x00
	commandDigitalPort     command = 0x01
	commandDigitalLatch    command = 0x02
	// Analog input commands
	commandAnalogInput       command = 0x10
	commandAnalogStartScan   command = 0x11
	commandAnalogStopScan    command = 0x12
	commandAnalogConfig      command = 0x14
	commandAnalogClearBuffer command = 0x15
	// Counter/timer commands
	commandEventCounter command = 0x20
	// Memory commands
	commandCalibrationMemory command = 0x30
	commandUserMemory        command = 0x31
	commandMBDMemory         command = 0x32
	// Miscellaneous commands
	commandBlinkLED        command = 0x41
	commandReset           command = 0x42
	commandGetStatus       command = 0x44
	commandSerialNum       command = 0x48
	commandUpgradeFirmware command = 0x50
	// Message-Based DAQ (MBD) Protocal commands
	commandTextMBD command = 0x80
	commandRawMBD  command = 0x81
)

var commands = map[command]string{
	commandDigitalTristate:   "Read/write tri-state register",
	commandDigitalPort:       "Read digital port pins",
	commandDigitalLatch:      "Read/write digital port output latch register",
	commandAnalogInput:       "Read analog input channel",
	commandAnalogStartScan:   "Start analog input scan",
	commandAnalogStopScan:    "Stop analog input scan",
	commandAnalogConfig:      "Configure the analog input channel",
	commandAnalogClearBuffer: "Clear the analog input scan FIFO buffer",
	commandEventCounter:      "Read/reset event counter",
	commandCalibrationMemory: "Read/write calibration memory",
	commandUserMemory:        "Read/write user memory",
	commandMBDMemory:         "Read/write Message-Based DAQ (MBD) memory",
	commandBlinkLED:          "Blink LED",
	commandReset:             "Reset device",
	commandGetStatus:         "Read device status",
	commandSerialNum:         "Read/write serial number",
	commandUpgradeFirmware:   "Enter device firmware upgrade (DFU) mode",
	commandTextMBD:           "Text-based MBD command/response",
	commandRawMBD:            "Raw MBD response",
}

func (c command) String() string {
	return commands[c]
}

type scanOption byte

// Analog input scan options
const (
	scanBlockTransferMode     scanOption = 0x0 << 0
	scanImmediateTransferMode scanOption = 0x1 << 0
	scanInternalPacerOff      scanOption = 0x0 << 1
	scanInternalPacerOn       scanOption = 0x1 << 1
	scanNoTrigger             scanOption = 0x0 << 2
	scanTriggerRisingEdge     scanOption = 0x1 << 2
	scanTriggerFallingEdge    scanOption = 0x2 << 2
	scanTriggerHighLevel      scanOption = 0x3 << 2
	scanTriggerLowLevel       scanOption = 0x4 << 2
	scanDebugMode             scanOption = 0x1 << 5
	scanStallOnOverrun        scanOption = 0x0 << 7
	scanInhibitStall          scanOption = 0x1 << 7
)

type analogInputSetup byte

// Analog input setup
const (
	singleEnded  analogInputSetup = 0
	differential analogInputSetup = 1
	calibration  analogInputSetup = 3
)

const (
	lastChannel               byte = 0x80
	maxBulkTransferPacketSize      = 64
)

type VoltageRange byte

// Ranges
const (
	Range10V VoltageRange = 0x0 // ±10V
	Range5V  VoltageRange = 0x1 // ±5V
	Range2V  VoltageRange = 0x3 // ±2V
	Range1V  VoltageRange = 0x5 // ±1V
)

// InputRanges maps the string keys that can be used in a JSON config file to
// the VoltageRange byte values.
var InputRanges = map[string]VoltageRange{
	"10V": Range10V,
	"5V":  Range5V,
	"2V":  Range2V,
	"1V":  Range1V,
}

type statusBit byte

// Status bit values
const (
	scanRunning statusBit = 0x1 << 1
	scanOverrun statusBit = 0x1 << 2
)

const (
	maxNumADChannels = 8  // max number of A/D channels in device
	maxNumGainLevels = 8  // max number of gain levels in device
	maxPacketSize    = 64 // max packet size for FS device
)
