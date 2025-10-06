# Serial Communication Library for Go

A clean, idiomatic Go library for serial port communication with advanced flow control support, designed for reliability and developer experience.

## Project Goals

- **Low-level CTS support**: Handle microsecond-precision CTS timing for devices with short CTS windows
- **Clean API**: Simple, intuitive interface that follows Go conventions
- **Flow control first**: Built-in support for hardware flow control patterns
- **Linux focus**: Optimized for Linux (x86_64 and ARM/Raspberry Pi)
- **Production ready**: Clean, DRY code following Go best practices

## Why Another Serial Library?

Existing Go serial libraries lack robust support for hardware flow control, particularly CTS (Clear To Send) timing. This library was created to handle devices that require precise timing for CTS windows as short as 480 microseconds.

## API Design

### Functional Options Pattern

Clean, discoverable API using functional options with sensible defaults:

```go
package main

import (
    "context"
    "time"
    "github.com/allbin/go-serial"
)

func main() {
    // Simple usage - all defaults
    port, err := serial.Open("/dev/ttyUSB0")
    if err != nil {
        panic(err)
    }
    defer port.Close()

    // With options - clear and readable
    port, err = serial.Open("/dev/ttyUSB0",
        serial.WithBaudRate(115200),
        serial.WithFlowControl(serial.FlowControlCTS),
        serial.WithInitialRTS(true),  // Required for CTS flow control
        serial.WithCTSTimeout(200*time.Millisecond),
    )
    if err != nil {
        panic(err)
    }

    // Context-based I/O with timeout control
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    n, err := port.WriteContext(ctx, []byte("Hello, Serial!"))
    if err != nil {
        panic(err)
    }

    // Simple blocking I/O (no timeout)
    buffer := make([]byte, 256)
    n, err = port.Read(buffer)
    if err != nil {
        panic(err)
    }
}
```

### Port Discovery

```go
// List available serial ports (ttyUSB*, ttyACM*, ttyS*, ttyAMA*)
ports, err := serial.ListPorts()
if err != nil {
    panic(err)
}
for _, port := range ports {
    fmt.Println("Found port:", port)
}
```

### USB Device Metadata (Linux)

Get detailed USB device information including vendor/product IDs, serial numbers, and interface details:

```go
// Get USB metadata for a specific port
info, err := serial.GetPortInfo("/dev/ttyUSB0")
if err != nil {
    panic(err)
}

fmt.Printf("Port: %s\n", info.Path)
fmt.Printf("  Vendor: %s Product: %s\n", info.VendorID, info.ProductID)
fmt.Printf("  Serial: %s Interface: %s\n", info.SerialNumber, info.InterfaceNumber)
fmt.Printf("  Manufacturer: %s\n", info.Manufacturer)
fmt.Printf("  Product: %s\n", info.Product)

// Iterate through all ports with metadata
ports, _ := serial.ListPorts()
for _, portPath := range ports {
    info, _ := serial.GetPortInfo(portPath)
    if info.SerialNumber != "" {
        fmt.Printf("%s: %s (Serial: %s)\n",
            info.Name, info.Description, info.SerialNumber)
    }
}
```

**Application-level device detection example:**

```go
// Find a specific device by USB metadata
func findDeviceByVendorAndProduct(vendorID, productID string) (string, error) {
    ports, _ := serial.ListPorts()
    for _, portPath := range ports {
        info, _ := serial.GetPortInfo(portPath)

        // Match by vendor and product ID
        if info.VendorID == vendorID && info.ProductID == productID {
            return portPath, nil
        }
    }
    return "", errors.New("device not found")
}

// Find a device by serial number pattern
func findDeviceBySerialPattern(prefix, suffix string) (string, error) {
    ports, _ := serial.ListPorts()
    for _, portPath := range ports {
        info, _ := serial.GetPortInfo(portPath)

        // Match by serial number pattern
        if strings.HasPrefix(info.SerialNumber, prefix) &&
           strings.HasSuffix(info.SerialNumber, suffix) {
            return portPath, nil
        }
    }
    return "", errors.New("device not found")
}
```

### USB Device Reset (Linux)

Programmatically reset USB devices to recover from hardware hangs:

```go
// Reset by port path
err := serial.ResetUSBDevice("/dev/ttyUSB0")
if err == serial.ErrUSBResetNotAvailable {
    log.Println("Install usbutils: sudo apt-get install usbutils")
}

// Reset by serial number (survives device re-enumeration)
err = serial.ResetUSBDeviceBySerial("FT123456")
if err != nil {
    log.Printf("Reset failed: %v", err)
}

// Check if reset is available before attempting
if serial.IsUSBResetAvailable() {
    err := serial.ResetUSBDevice("/dev/ttyUSB0")
    // handle error
}
```

**Requirements:**
- `usbreset` utility from `usbutils` package
- Root/sudo permissions for USB operations

**Note:** USB devices re-enumerate after reset, potentially changing their ttyUSB number. Use serial numbers for reliable device identification after reset.

### Modem Signal Control and Monitoring

Access and control modem control signals (RTS, DTR, CTS, DSR, RI, DCD) for hardware flow control and device signaling:

```go
// Read all modem signal states
signals, err := port.GetModemSignals()
if err != nil {
    panic(err)
}

fmt.Printf("CTS: %v, DSR: %v, DCD: %v, RI: %v\n",
    signals.CTS, signals.DSR, signals.DCD, signals.RI)

// Manual RTS control for software flow control
err = port.SetRTS(false)  // Signal not ready
// Process buffer
err = port.SetRTS(true)   // Signal ready

// Check RTS state
rtsHigh, err := port.GetRTS()

// Wait for signal changes (event-driven monitoring)
signals, changed, err := port.WaitForSignalChange(
    serial.SignalDSR | serial.SignalDCD,
    5*time.Second,
)
if err == serial.ErrSignalTimeout {
    // No signal change within timeout
}
if changed&serial.SignalDCD != 0 && !signals.DCD {
    // Active-low wake signal detected
}

// Context-aware signal monitoring
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

signals, changed, err = port.WaitForSignalChangeContext(ctx, serial.SignalDSR)
```

**Initial signal configuration:**

```go
// Set initial RTS/DTR states when opening port
port, err := serial.Open("/dev/ttyUSB0",
    serial.WithInitialRTS(true),   // Start with RTS asserted
    serial.WithInitialDTR(true),   // Start with DTR asserted
)
```

**Use cases:**
- Wake-up signals (active-low DSR/DCD patterns)
- Device ready indicators (DSR)
- External event triggers (RI - Ring Indicator)
- Software flow control (manual RTS control)

### Available Options

```go
// Configuration options
serial.WithBaudRate(115200)
serial.WithDataBits(8)              // 5, 6, 7, 8
serial.WithStopBits(1)              // 1, 2
serial.WithParity(serial.ParityEven) // None, Odd, Even, Mark, Space
serial.WithFlowControl(serial.FlowControlCTS) // None, CTS, RTSCTS (requires WithInitialRTS)
serial.WithCTSTimeout(10*time.Second)
serial.WithReadTimeout(25)          // VTIME in tenths of seconds (0-255)
serial.WithWriteMode(serial.WriteModeSynced) // Buffered, Synced
serial.WithSyncWrite()              // Shorthand for synced writes
serial.WithInitialRTS(true)         // Set initial RTS state (required for flow control)
serial.WithInitialDTR(true)         // Set initial DTR state
```

### Default Configuration

- **BaudRate**: 115200
- **DataBits**: 8
- **StopBits**: 1
- **Parity**: None
- **FlowControl**: None
- **CTSTimeout**: 10s
- **ReadTimeout**: 2.5 seconds (25 tenths)
- **WriteMode**: Buffered

### Error Handling

Specific error types for robust error handling with `errors.Is()`:

```go
var (
    ErrCTSTimeout           = errors.New("CTS timeout waiting for clear to send")
    ErrInvalidBaudRate      = errors.New("invalid baud rate")
    ErrInvalidConfig        = errors.New("invalid serial configuration")
    ErrPortClosed           = errors.New("serial port is closed")
    ErrSignalTimeout        = errors.New("timeout waiting for signal change")
    ErrInvalidSignalMask    = errors.New("invalid signal mask")
    ErrUSBInfoNotAvailable  = errors.New("USB device information not available")
    ErrUSBResetNotAvailable = errors.New("usbreset utility not available")
)

// Usage
if err := port.Write(data); errors.Is(err, serial.ErrCTSTimeout) {
    // Handle CTS timeout specifically
}
if errors.Is(err, serial.ErrSignalTimeout) {
    // Handle signal monitoring timeout
}
```

### Platform Support

**Core Serial Communication:** Works on all Linux systems (x86_64, ARM, Raspberry Pi)

**USB Features (Linux-only):**
- USB device metadata extraction relies on Linux sysfs (`/sys/class/tty/`)
- USB device reset requires `usbreset` utility from `usbutils` package
- On non-Linux platforms, USB metadata fields return empty strings
- USB reset functions return `ErrUSBInfoNotAvailable` on non-Linux systems

### Hardware-Specific Considerations

#### Neocortec Mesh Network Modules

Neocortec modules use **scheduled event-based CTS signaling** rather than continuous CTS availability:

**CTS Timing Characteristics:**
- CTS window: Default 488 microseconds (16 units x 30.5us, configurable 1-255 via AAPI ID 51)
- CTS only activates during TX Scheduled Data events
- Events occur periodically based on Scheduled Data Rate configuration
- Between events, the module may sleep for power conservation
- **Note**: CTS signaling follows standard UART conventions (TIOCM_CTS bit set = ready to send)

**Configuration Requirements:**
```go
port, err := serial.Open("/dev/ttyUSB0",
    serial.WithBaudRate(115200),              // Neocortec default: 115200, 8N1
    serial.WithFlowControl(serial.FlowControlCTS),
    serial.WithInitialRTS(true),              // Assert RTS on port open
    serial.WithCTSTimeout(10*time.Second),    // Worst-case: wait for next scheduled event
)
```

**Why Large Timeout is Needed:**
- CTS activates only during scheduled data events (potentially seconds apart)
- Missing the 488us window requires waiting for next event
- Default 10s timeout accommodates worst-case scheduled data periods
- For faster response, configure module's CTS Interleave (AAPI ID 50) to trigger on every Wake Up event

**Implementation Details:**
- Write operations are **pre-queued** before CTS goes LOW
- When CTS activates (goes LOW), data is written **immediately** with no scheduling delay
- This ensures transmission begins within the 488us CTS window
- Pattern matches Neocortec's reference implementation for maximum reliability

**Troubleshooting:**
- First message works, subsequent fail: Likely missing CTS windows between scheduled events
- Consistent failures: Check CTS polarity, module configuration, or physical connections
- Monitor CTS events: `serial monitor /dev/ttyUSB0 --signals cts`

**Module Configuration (via System UART):**
- AAPI CTS Timeout (ID 51): Increase for longer host transmission windows
- AAPI CTS Interleave (ID 50): Set to 0 for more frequent CTS events
- Scheduled Data Rate: Adjust event frequency for application needs

**Physical Interface:**
- **CTS (pin 19/4)**: Module output, asserted when ready to receive data
- **RTS**: Not used by Neocortec for flow control (can be asserted high during initialization)
- **UART**: 115200 baud, 8N1, no parity

#### Other Scheduled Event-Based Devices

Similar considerations apply to any device using event-driven CTS signaling:
- Set CTSTimeout to accommodate event period (not just CTS pulse width)
- Monitor CTS activity to understand device timing pattern
- Consider device sleep/wake cycles when planning communication

## Architecture Principles

### Developer Experience Focus

- **Transparent flow control**: `Write()` method handles all flow control logic internally
- **Blocking semantics**: Simple, predictable behavior - blocks until operation completes
- **Configuration-driven**: Behavior controlled through config, not API complexity
- **Standard interfaces**: Implements `io.ReadWriteCloser` for compatibility

### Implementation Strategy

- **Unix package integration**: Clean golang.org/x/sys/unix usage for all ioctl operations
- **Direct file descriptor**: Using unix.Open() for precise control and reliable operation
- **TIOCM support**: Hardware modem control signals for CTS monitoring
- **Clean termios configuration**: Proper 8N1 raw mode setup with configurable parameters
- **Clean error handling**: Descriptive errors with context

## Implementation Status

### Core Library - COMPLETED ✅

- [x] **Serial Port Operations**: Clean, reliable UART communication with unix package integration
- [x] **Flow Control**: Hardware CTS/RTS support with configurable timeouts
- [x] **Configuration System**: Functional options pattern with comprehensive validation
- [x] **Port Discovery**: Automatic detection and filtering of communication devices
- [x] **USB Device Metadata**: Extract vendor/product IDs, serial numbers, interface details (Linux)
- [x] **USB Device Reset**: Programmatic USB reset for hung devices (Linux)
- [x] **Modem Signal Control**: Full modem signal monitoring and control (CTS, DSR, RI, DCD, RTS, DTR)
- [x] **Error Handling**: Proper error types with context-aware messaging
- [x] **Testing**: Unit tests covering configuration, I/O operations, USB features, modem signals, and edge cases

### CLI Tool - COMPLETED ✅

Professional command-line interface with interactive features:

- [x] **Port Management**: `serial list` with filtering and USB metadata in table view
- [x] **USB Device Info**: `serial info` displays detailed USB device information
- [x] **USB Device Reset**: `serial reset` for recovering hung USB devices
- [x] **Modem Signal Control**: `serial signals`, `serial monitor`, `serial rts`, `serial dtr` for signal monitoring and control
- [x] **Data Communication**: `serial send` and `serial listen` for basic I/O
- [x] **Interactive Terminal**: `serial connect` with real-time bidirectional communication
- [x] **Flow Control Support**: Hardware CTS/RTS support with configurable timeouts
- [x] **Advanced Features**: Synchronous writes, hex mode, timeout control

### Future Enhancements

- [ ] **Advanced Hardware Support**: Custom baud rates, break signals
- [ ] **Performance Optimizations**: Zero-copy I/O, interrupt-driven signal monitoring
- [ ] **Platform Extensions**: Windows support, additional embedded platforms
- [ ] **Additional Signal Features**: Line status monitoring (overrun, framing, parity errors)

## CLI Tool Usage

```bash
# Port discovery and management
serial list                           # List available ports
serial list --table --filter usb     # Styled table with USB metadata
serial info /dev/ttyUSB0             # Show detailed USB device info

# USB device management
sudo serial reset /dev/ttyUSB0       # Reset USB device by port
sudo serial reset --serial FT123456  # Reset USB device by serial number

# Modem signal control and monitoring
serial signals /dev/ttyUSB0          # Display current signal states
serial monitor /dev/ttyUSB0          # Monitor signal changes
serial monitor /dev/ttyUSB0 --signals cts,dsr  # Monitor specific signals
serial rts /dev/ttyUSB0 high         # Set RTS high
serial rts /dev/ttyUSB0 low          # Set RTS low
serial dtr /dev/ttyUSB0 high         # Set DTR high

# Data communication
serial listen /dev/ttyUSB0           # Real-time data monitoring
serial send "Hello World" /dev/ttyUSB0  # Send data to port
echo "test" | serial send /dev/ttyUSB0   # Pipe data to port

# Interactive terminal
serial connect /dev/ttyUSB0          # Bidirectional communication
serial connect /dev/ttyUSB0 --flow-control cts --initial-rts
serial connect /dev/ttyUSB0 --sync-writes --flow-control cts --initial-rts

# Connect UI features:
# - Real-time TX status tracking: ENQUEUED → SENT (with timing in ms)
# - Visual feedback: Yellow (enqueued), Green (sent), Orange (timeout), Red (error)
# - CTS flow control timing visibility for debugging
# - Timeout messages show "MAY STILL SEND" (queued writes may complete after timeout)
```

#### Repository Structure

**Library-first design** with clean import path and standard Go project layout:

```
serial/
├── cmd/                     # CLI commands (Cobra)
│   ├── connect.go           # Interactive terminal connection
│   ├── info.go              # USB device information display
│   ├── list.go              # Port discovery and listing
│   ├── listen.go            # Real-time data monitoring
│   ├── reset.go             # USB device reset
│   ├── send.go              # Send data to port
│   └── root.go              # CLI root configuration
├── cmd/serial/              # CLI application entry point
│   └── main.go              # package main
├── internal/                # CLI-specific code (unexported)
│   └── tui/                 # Bubble Tea TUI components
├── port.go                  # Core serial port implementation
├── config.go                # Configuration and functional options
├── errors.go                # Error types and definitions
├── list.go                  # Port discovery and USB metadata
├── usb_reset.go             # USB device reset functionality
├── port_test.go             # Unit tests
├── list_test.go             # Port discovery tests
├── usb_test.go              # USB feature tests
├── go.mod
└── README.md
```

**Design Benefits:**

- **Clean import path**: `github.com/allbin/serial` (no pkg/ subdirectory)
- **Standard Go layout**: Library in root, CLI in `cmd/`
- **No circular dependencies**: One-way dependency (CLI → Library)
- **Professional structure**: Follows Go project best practices

## Installation

```bash
# Library installation
go get github.com/allbin/go-serial

# CLI tool installation
go install github.com/allbin/go-serial/cmd/serial@latest

# Development usage
go run ./cmd/serial list --table --filter usb
go run ./cmd/serial connect /dev/ttyUSB0 --flow-control cts --initial-rts
```

---

_A clean, reliable Go library for serial communication with advanced flow control support._

