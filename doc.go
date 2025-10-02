// Package serial provides a clean, idiomatic Go library for serial port communication
// with advanced hardware flow control support.
//
// This library is designed for reliable serial communication on Linux systems (x86_64 and ARM),
// with emphasis on precise CTS (Clear To Send) timing for devices requiring microsecond-precision
// flow control windows.
//
// # Basic Usage
//
// Open a serial port with default configuration (115200 8N1, no flow control):
//
//	port, err := serial.Open("/dev/ttyUSB0")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer port.Close()
//
//	// Simple I/O
//	n, err := port.Write([]byte("Hello"))
//	buffer := make([]byte, 256)
//	n, err = port.Read(buffer)
//
// # Configuration Options
//
// Use functional options for custom configuration:
//
//	port, err := serial.Open("/dev/ttyUSB0",
//	    serial.WithBaudRate(115200),
//	    serial.WithFlowControl(serial.FlowControlCTS),
//	    serial.WithCTSTimeout(200*time.Millisecond),
//	    serial.WithInitialRTS(true),
//	    serial.WithInitialDTR(true),
//	)
//
// # Port Discovery
//
// List available serial ports and get USB device metadata:
//
//	ports, err := serial.ListPorts()
//	for _, portPath := range ports {
//	    info, _ := serial.GetPortInfo(portPath)
//	    fmt.Printf("%s: %s (VID=%s PID=%s Serial=%s)\n",
//	        info.Path, info.Description, info.VendorID, info.ProductID, info.SerialNumber)
//	}
//
// # Hardware Flow Control
//
// Monitor and control modem signals (CTS, DSR, DCD, RI, RTS, DTR):
//
//	// Read signal states
//	signals, err := port.GetModemSignals()
//	fmt.Printf("CTS=%v DSR=%v DCD=%v RI=%v\n",
//	    signals.CTS, signals.DSR, signals.DCD, signals.RI)
//
//	// Control RTS/DTR
//	err = port.SetRTS(true)
//	err = port.SetDTR(false)
//
//	// Wait for signal changes (event-driven)
//	signals, changed, err := port.WaitForSignalChange(
//	    serial.SignalDSR|serial.SignalDCD,
//	    5*time.Second,
//	)
//
// # USB Device Management (Linux)
//
// Reset hung USB devices programmatically:
//
//	// Reset by port path
//	err := serial.ResetUSBDevice("/dev/ttyUSB0")
//
//	// Reset by serial number (survives re-enumeration)
//	err = serial.ResetUSBDeviceBySerial("FT123456")
//
// Requires usbreset utility from usbutils package and root/sudo permissions.
//
// # Context Support
//
// All I/O operations support context for timeout and cancellation control:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	n, err := port.WriteContext(ctx, data)
//	n, err = port.ReadContext(ctx, buffer)
//
// # Error Handling
//
// The library provides specific error types for robust error handling:
//
//	var (
//	    ErrCTSTimeout           // CTS flow control timeout
//	    ErrPortClosed           // Port already closed
//	    ErrSignalTimeout        // Signal change timeout
//	    ErrUSBInfoNotAvailable  // USB metadata unavailable
//	    ErrUSBResetNotAvailable // usbreset utility not found
//	    // ... and more
//	)
//
// Use errors.Is() for error type checking:
//
//	if errors.Is(err, serial.ErrCTSTimeout) {
//	    // Handle CTS timeout specifically
//	}
//
// # Platform Support
//
// Core serial communication works on all Linux systems. USB-specific features
// (metadata extraction, device reset) are Linux-only and rely on sysfs and
// the usbreset utility.
//
// # Default Configuration
//
//   - BaudRate: 115200
//   - DataBits: 8
//   - StopBits: 1
//   - Parity: None
//   - FlowControl: None
//   - CTSTimeout: 500ms
//   - ReadTimeout: 2.5 seconds
//   - WriteMode: Buffered
//
// For more details and advanced usage examples, see the README at:
// https://github.com/allbin/go-serial
package serial
