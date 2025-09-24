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
    "github.com/allbin/serial"
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
serial.WithReadTimeout(25)          // VTIME in tenths of seconds (0-255)
serial.WithWriteMode(serial.WriteModeSynced) // Buffered, Synced
serial.WithSyncWrite()              // Shorthand for synced writes
```

### Default Configuration

- **BaudRate**: 115200
- **DataBits**: 8
- **StopBits**: 1
- **Parity**: None
- **FlowControl**: None
- **CTSTimeout**: 500ms
- **ReadTimeout**: 2.5 seconds (25 tenths)
- **WriteMode**: Buffered

### Error Handling

Specific error types for robust error handling with `errors.Is()`:

```go
var (
    ErrCTSTimeout        = errors.New("CTS timeout waiting for clear to send")
    ErrInvalidBaudRate   = errors.New("invalid baud rate")
    ErrInvalidConfig     = errors.New("invalid serial configuration")
    ErrPortClosed        = errors.New("serial port is closed")
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
- [x] **Error Handling**: Proper error types with context-aware messaging
- [x] **Testing**: Unit tests covering configuration, I/O operations, and edge cases

### CLI Tool - COMPLETED ✅

Professional command-line interface with interactive features:

- [x] **Port Management**: `serial list` with filtering and table formatting
- [x] **Data Communication**: `serial send` and `serial listen` for basic I/O
- [x] **Interactive Terminal**: `serial connect` with real-time bidirectional communication
- [x] **Flow Control Support**: Hardware CTS/RTS support with configurable timeouts
- [x] **Advanced Features**: Synchronous writes, hex mode, timeout control

### Future Enhancements

- [ ] **Advanced Hardware Support**: Custom baud rates, break signals, additional modem control
- [ ] **Performance Optimizations**: Zero-copy I/O, interrupt-driven CTS monitoring
- [ ] **Platform Extensions**: Windows support, additional embedded platforms

## CLI Tool Usage

```bash
# Port discovery and management
serial list                           # List available ports
serial list --table --filter usb     # Styled table with USB filtering

# Data communication
serial listen /dev/ttyUSB0           # Real-time data monitoring
serial send "Hello World" /dev/ttyUSB0  # Send data to port
echo "test" | serial send /dev/ttyUSB0   # Pipe data to port

# Interactive terminal
serial connect /dev/ttyUSB0          # Bidirectional communication
serial connect /dev/ttyUSB0 --sync-writes --flow-control cts
```

#### Repository Structure

**Library-first design** with clean import path and standard Go project layout:

```
serial/
├── cmd/                     # CLI commands (Cobra)
│   ├── connect.go           # Interactive terminal connection
│   ├── list.go              # Port discovery and listing
│   ├── listen.go            # Real-time data monitoring
│   ├── send.go              # Send data to port
│   └── root.go              # CLI root configuration
├── cmd/serial/              # CLI application entry point
│   └── main.go              # package main
├── internal/                # CLI-specific code (unexported)
│   └── tui/                 # Bubble Tea TUI components
├── port.go                  # Core serial port implementation
├── config.go                # Configuration and functional options
├── errors.go                # Error types and definitions
├── list.go                  # Port discovery functionality
├── port_test.go             # Unit tests
├── list_test.go             # Port discovery tests
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
go get github.com/allbin/serial

# CLI tool installation
go install github.com/allbin/serial/cmd/serial@latest

# Development usage
go run ./cmd/serial list --table --filter usb
go run ./cmd/serial connect /dev/ttyUSB0 --flow-control cts
```

---

_A clean, reliable Go library for serial communication with advanced flow control support._

