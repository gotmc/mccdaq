package usb1608fsplus

type command int

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
	// Miscellaneous commands
	commandBlinkLED  command = 0x41
	commandReset     command = 0x42
	commandGetStatus command = 0x44
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
	commandBlinkLED:          "Blink LED",
	commandReset:             "Reset device",
	commandGetStatus:         "Read device status",
}

func (c command) String() string {
	return commands[c]
}
