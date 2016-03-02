package usb1608fsplus

import (
	"encoding/binary"
	"fmt"

	"github.com/gotmc/libusb"
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

// ScanAnalogInput starts an analog input scan. If an AInScan is currently
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
   provide the pacer sourece.  The A/Ds acquire data on every rising
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
