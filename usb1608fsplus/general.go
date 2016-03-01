package usb1608fsplus

import (
	"encoding/binary"
	"log"

	"github.com/gotmc/libusb"
)

func BlinkLED(dh *libusb.DeviceHandle, count int) error {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice,
		libusb.Vendor,
		libusb.DeviceRecipient,
	)
	timeout := 20

	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, uint16(uint8(count)))
	log.Printf("b = 0x%x\n", b)

	count, err := dh.ControlTransfer(
		requestType,
		byte(commandBlinkLED),
		0x0,
		0x0,
		b,
		len(b),
		timeout,
	)
	if err != nil {
		log.Printf("Error %s\n", err)
		return err
	}
	return nil
}

func Reset(dh *libusb.DeviceHandle) error {
	requestType := libusb.BitmapRequestType(
		libusb.HostToDevice,
		libusb.Vendor,
		libusb.DeviceRecipient,
	)
	log.Printf("bmRequestType = 0x%x\n", requestType)
	// Reset = 0x42
	data := []byte{0x42}
	log.Printf("data = %v\n", data)
	timeout := 20

	_, err := dh.ControlTransfer(
		requestType,
		byte(0x42),
		0x0,
		0x0,
		[]byte{0x00},
		1,
		timeout,
	)
	if err != nil {
		log.Printf("Error %s\n", err)
		return err
	}
	return nil
}
