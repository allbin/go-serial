package serial

import (
	"fmt"
	"os/exec"
	"time"
)

// ResetUSBDevice performs a USB-level reset of the device
// This can recover hardware that is in a hung/unresponsive state
//
// Requirements:
// - usbreset utility must be installed (from usbutils package)
// - Requires appropriate permissions (typically root/sudo)
//
// Returns:
// - nil if reset successful
// - ErrUSBResetNotAvailable if usbreset utility not found
// - ErrUSBInfoNotAvailable if device is not USB or metadata unavailable
// - error if reset fails
func ResetUSBDevice(portPath string) error {
	// Get port info to find USB bus/device numbers
	info, err := GetPortInfo(portPath)
	if err != nil {
		return fmt.Errorf("failed to get port info: %w", err)
	}

	if info.BusNumber == "" || info.DeviceNumber == "" {
		return ErrUSBInfoNotAvailable
	}

	// Check if usbreset utility is available
	if !IsUSBResetAvailable() {
		return ErrUSBResetNotAvailable
	}

	// Construct USB device path in format expected by usbreset (BBB/DDD)
	// usbreset expects zero-padded 3-digit bus and device numbers
	usbPath := fmt.Sprintf("%03s/%03s", info.BusNumber, info.DeviceNumber)

	// Execute USB reset
	cmd := exec.Command("usbreset", usbPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("usbreset failed: %w (output: %s)", err, string(output))
	}

	// Wait for device to re-enumerate
	// USB devices typically take 1-2 seconds to become available again
	time.Sleep(2 * time.Second)

	return nil
}

// ResetUSBDeviceBySerial resets a USB device by its serial number
// Useful when device paths change after reboot or when multiple devices are connected
func ResetUSBDeviceBySerial(serialNumber string) error {
	ports, err := ListPorts()
	if err != nil {
		return err
	}

	for _, portPath := range ports {
		info, err := GetPortInfo(portPath)
		if err != nil {
			continue
		}

		if info.SerialNumber == serialNumber {
			return ResetUSBDevice(portPath)
		}
	}

	return fmt.Errorf("device with serial %s not found", serialNumber)
}

// IsUSBResetAvailable checks if usbreset utility is available in PATH
func IsUSBResetAvailable() bool {
	_, err := exec.LookPath("usbreset")
	return err == nil
}
