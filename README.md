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
    "github.com/mdjarv/serial"
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

### Available Options
```go
// Configuration options
serial.WithBaudRate(115200)
serial.WithDataBits(8)              // 5, 6, 7, 8
serial.WithStopBits(1)              // 1, 2
serial.WithParity(serial.ParityEven) // None, Odd, Even, Mark, Space
serial.WithFlowControl(serial.FlowControlCTS) // None, CTS, RTSCTS
serial.WithCTSTimeout(500*time.Millisecond)
serial.WithReadBufferSize(4096)
serial.WithWriteBufferSize(4096)
```

### Default Configuration
- **BaudRate**: 115200
- **DataBits**: 8
- **StopBits**: 1
- **Parity**: None
- **FlowControl**: None
- **CTSTimeout**: 500ms
- **BufferSizes**: 4096 bytes

### Error Handling
Specific error types for robust error handling with `errors.Is()`:

```go
var (
    ErrDeviceNotFound    = errors.New("serial device not found")
    ErrPermissionDenied  = errors.New("permission denied accessing serial device")
    ErrDeviceInUse       = errors.New("serial device already in use")
    ErrCTSTimeout        = errors.New("CTS timeout waiting for clear to send")
    ErrInvalidBaudRate   = errors.New("invalid baud rate")
    ErrInvalidConfig     = errors.New("invalid serial configuration")
)

// Usage
if err := port.Write(data); errors.Is(err, serial.ErrCTSTimeout) {
    // Handle CTS timeout specifically
}
```

## Architecture Principles

### Developer Experience Focus
- **Transparent flow control**: `Write()` method handles all flow control logic internally
- **Blocking semantics**: Simple, predictable behavior - blocks until operation completes
- **Configuration-driven**: Behavior controlled through config, not API complexity
- **Standard interfaces**: Implements `io.ReadWriteCloser` for compatibility

### Implementation Strategy
- **Linux termios**: Direct system calls for precise control
- **TIOCM support**: Use modem control signals for CTS monitoring
- **Interrupt-driven**: React to CTS changes without polling where possible
- **Clean error handling**: Descriptive errors with context

## Implementation Status

### Completed Features
- [x] **API Design**: Functional options pattern with sensible defaults
- [x] **Architecture Planning**: Context-based I/O, flow control strategy
- [x] **Documentation**: Comprehensive API examples and usage patterns
- [x] **Project Structure**: Directory structure created following Go best practices
- [x] **Core Port Types**: Port struct, FlowControl, and Parity enums defined (port.go:14-41)
- [x] **Termios Integration**: Clean implementation using golang.org/x/sys/unix instead of magic numbers
- [x] **Baud Rate Support**: Complete baud rate mapping with validation (port.go:43-109)
- [x] **Low-level Termios Helpers**: setTermios, getTermios, getModemStatus functions (port.go:111-137)

### Core Features

#### Basic Serial Communication - COMPLETED ✅

**MAJOR MILESTONE ACHIEVED:**
All core serial communication functionality has been implemented and tested successfully!

**Completed Implementation:**
- [x] **Port Opening & Configuration** (100% complete)
  - [x] Linux termios integration foundation
  - [x] Baud rate, data bits, stop bits, parity configuration logic
  - [x] Termios helper functions (setTermios, getTermios, getModemStatus)
  - [x] Complete Open() function implementation with termios configuration (port.go:139-265)
  - [x] Functional options implementation (config.go:35-114) - all WithXXX() functions
  - [x] Error handling integration (errors.go:7-16) - comprehensive error types
- [x] **Basic I/O Operations** (100% complete)
  - [x] `Read()` and `Write()` methods (blocking) (port.go:287-313)
  - [x] `ReadContext()` and `WriteContext()` with timeout support (port.go:334-462)
  - [x] Advanced CTS flow control with microsecond precision (port.go:415-462)
  - [x] Proper resource cleanup and `Close()` implementation (port.go:268-284)
- [x] **Configuration System** (config.go:6-114)
  - [x] Config struct with all serial parameters
  - [x] DefaultConfig() with sensible defaults (115200, 8N1, no flow control)
  - [x] All functional options: WithBaudRate, WithDataBits, WithStopBits, WithParity, WithFlowControl, WithCTSTimeout, buffer sizing
- [x] **Error Handling** (errors.go:7-16)
  - [x] Comprehensive error types for device access, configuration, timeouts, and flow control
  - [x] Proper error wrapping and context in all operations
- [x] **Testing** (port_test.go)
  - [x] Unit tests for configuration system
  - [x] Functional options validation
  - [x] Error condition testing
  - [x] Context timeout behavior verification
  - [x] All tests passing (8/8 ✅)

**NEXT PRIORITIES:**
1. **CLI Tool**: Start building the Cobra-based command-line interface (`serial list`, `serial connect`)
2. **Advanced Flow Control**: Enhanced CTS timeout and interrupt-driven monitoring
3. **Hardware Testing**: Test with real serial devices and CTS timing validation
4. **USB Device Info**: Complete sysfs parsing for USB vendor/product information

#### Flow Control System - PARTIAL IMPLEMENTATION ⚠️
- [x] **Hardware Flow Control** (basic implementation complete)
  - [x] `FlowControlNone` - Standard serial communication
  - [x] `FlowControlCTS` - CTS-aware transmission with configurable timeout
  - [x] `FlowControlRTSCTS` - Full hardware handshaking (uses kernel CRTSCTS)
- [x] **CTS Implementation** (basic version complete)
  - [x] Basic TIOCM integration for CTS signal monitoring (port.go:130-137)
  - [x] Configurable CTS timeout handling (port.go:415-462)
  - [x] Context vs CTS timeout precedence (first timeout wins)

**Flow Control Limitations & Future Work:**
- Current CTS checking is polling-based (10μs intervals) - could be optimized with interrupt-driven approach
- Need hardware testing to validate microsecond precision requirements
- Mid-transmission CTS handling needs validation and documentation

#### Port Discovery - COMPLETED ✅
- [x] **Device Enumeration** (100% complete)
  - [x] `ListPorts()` function with comprehensive device filtering (list.go:13-82)
  - [x] Filter for communication-capable devices (`ttyUSB*`, `ttyACM*`, `ttyS*`, `ttyAMA*`, and more)
  - [x] Exclude virtual terminals (`tty[nn]`, `console`, `ptmx`, etc.)
  - [x] Support for ARM (`ttyAMA*`), i.MX (`ttymxc*`), OMAP (`ttyO*`), Samsung (`ttySAC*`), Tegra (`ttyTHS*`)
  - [x] Character device validation and sorted output
- [x] **Port Information** (PortInfo struct and GetPortInfo function)
  - [x] Human-readable device descriptions based on port type
  - [x] Port path and name extraction
  - [x] USB device information framework (expandable for sysfs parsing)
- [x] **Testing** (list_test.go)
  - [x] Pattern matching validation
  - [x] Device filtering logic testing
  - [x] Integration testing with real system devices
  - [x] All tests passing (13/13 ✅)

### Research & Testing Phase

#### Hardware Testing
- [ ] **CTS Timing Validation**
  - [ ] Test with devices requiring microsecond CTS precision
  - [ ] Validate 480µs window handling capability
  - [ ] Performance benchmarking for CTS response times
- [ ] **Mid-transmission Behavior**
  - [ ] Test CTS signal changes during data transmission
  - [ ] Document expected behavior and error handling
  - [ ] Implement appropriate recovery strategies

#### Platform Compatibility
- [ ] **Linux Variants**
  - [ ] x86_64 desktop/server systems
  - [ ] ARM/Raspberry Pi GPIO serial (`ttyAMA*`, `ttyS0`)
  - [ ] USB serial adapter compatibility (`ttyUSB*`, `ttyACM*`)

### Future Enhancements
- [ ] **Advanced Features**
  - [ ] Concurrent read/write patterns (research needed)
  - [ ] Custom baud rate support
  - [ ] Break signal handling
  - [ ] Modem control signals (DTR, DSR, etc.)
- [ ] **Performance Optimizations**
  - [ ] Zero-copy I/O where possible
  - [ ] Optimized buffer management
  - [ ] Interrupt-driven CTS monitoring

### CLI Tool

#### Command-Line Interface
A professional CLI tool using Cobra + Bubble Tea for both library testing and general serial communication:

```bash
# Port discovery and management
serial list                           # Beautiful table of available ports
serial info /dev/ttyUSB0             # Detailed port information and status

# Data communication
serial listen /dev/ttyUSB0           # Real-time data monitoring with styled output
serial send "Hello World" /dev/ttyUSB0  # Send data to port
echo "test" | serial send /dev/ttyUSB0   # Pipe data to port

# Interactive features
serial connect                       # Interactive port selection + connection
serial terminal /dev/ttyUSB0        # Full-duplex terminal with TUI interface

# Testing and debugging utilities
serial test-cts /dev/ttyUSB0        # CTS timing validation and diagnostics
```

#### Repository Structure
**Library-first design** with clean import path and standard Go project layout:

```
serial/
├── cmd/serial/              # CLI application entry point
│   └── main.go              # package main
├── internal/                # CLI-specific code (unexported)
│   ├── commands/            # Cobra command implementations
│   ├── tui/                 # Bubble Tea TUI components
│   └── flags/               # Common flag definitions
├── examples/                # Library usage examples
├── port.go                  # Library code (package serial)
├── config.go                # Configuration and functional options
├── errors.go                # Error types and definitions
├── list.go                  # Port discovery functionality
├── go.mod
└── README.md
```

**Design Benefits:**
- **Clean import path**: `github.com/mdjarv/serial` (no pkg/ subdirectory)
- **Standard Go layout**: Library in root, CLI in `cmd/`
- **No circular dependencies**: One-way dependency (CLI → Library)
- **Professional structure**: Follows Go project best practices

#### CLI Implementation Roadmap
- [x] **Basic CLI Structure (Cobra)** - COMPLETED ✅
  - [x] Project initialization with `cobra-cli`
  - [x] `list` command with styled table output and filtering options
  - [x] Flag parsing and configuration (filter, table format)
  - [x] Integration with core library (ListPorts, GetPortInfo)
  - [ ] Additional commands (`info`, `send`, `listen`, `connect`, `terminal`)
- [ ] **Terminal UI Features (Bubble Tea + Charm)**
  - [ ] Interactive port selection with styled components
  - [ ] Real-time data display with hex/ASCII formatting
  - [ ] Split-pane terminal interface for send/receive
  - [ ] Status bars and connection indicators
  - [ ] Beautiful styling with `lipgloss`
- [ ] **Advanced CLI Features**
  - [ ] Configuration file support
  - [ ] Data logging and capture
  - [ ] Protocol analyzers and parsers
  - [ ] Scripting and automation support

#### Installation & Distribution
```bash
# Library installation
go get github.com/mdjarv/serial

# CLI tool installation
go install github.com/mdjarv/serial/cmd/serial@latest

# Usage (Development)
go run ./cmd/serial list --table --filter usb
go run ./cmd/serial list --help

# Usage (After Installation)
serial list --table --filter usb
serial connect /dev/ttyUSB0 --baudrate=115200 --flow-control=cts
```

### Development Workflow
- [ ] **Project Setup**
  - [x] Go module initialization
  - [x] Cobra CLI initialization with `cobra-cli init`
  - [ ] CI/CD pipeline setup
  - [ ] Testing framework integration
- [ ] **Documentation**
  - [ ] Go doc comments for all public APIs
  - [ ] CLI command documentation and examples
  - [ ] Usage examples and tutorials
  - [ ] Hardware-specific integration guides

## Development Philosophy

- **Clean, DRY code**: Follow Go best practices and idioms
- **Test-driven**: Comprehensive testing including hardware-in-the-loop
- **Documentation first**: Clear examples and usage patterns
- **Performance**: Optimize for low-latency, high-reliability communication

---

*This library is designed for applications requiring reliable hardware flow control while maintaining a simple, idiomatic Go API.*