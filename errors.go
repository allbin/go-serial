package serial

import "errors"

// Predefined error types for robust error handling
var (
	ErrDeviceNotFound   = errors.New("serial device not found")
	ErrPermissionDenied = errors.New("permission denied accessing serial device")
	ErrDeviceInUse      = errors.New("serial device already in use")
	ErrCTSTimeout       = errors.New("CTS timeout waiting for clear to send")
	ErrInvalidBaudRate  = errors.New("invalid baud rate")
	ErrInvalidConfig    = errors.New("invalid serial configuration")
	ErrPortClosed       = errors.New("serial port is closed")
	ErrWriteTimeout     = errors.New("write operation timed out")
	ErrReadTimeout      = errors.New("read operation timed out")

	// Signal monitoring errors
	ErrSignalTimeout     = errors.New("timeout waiting for signal change")
	ErrInvalidSignalMask = errors.New("invalid signal mask")

	// USB-related errors
	ErrUSBInfoNotAvailable  = errors.New("USB device information not available")
	ErrUSBResetNotAvailable = errors.New("usbreset utility not available")
)
