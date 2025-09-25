package serial

import (
	"context"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Port represents a serial port connection
type Port struct {
	mu           sync.RWMutex
	file         *os.File
	config       Config
	originalTerm unix.Termios
	closed       bool
}

// FlowControl represents the flow control mode
type FlowControl int

const (
	FlowControlNone FlowControl = iota
	FlowControlCTS
	FlowControlRTSCTS
)

// Parity represents the parity mode
type Parity int

const (
	ParityNone Parity = iota
	ParityOdd
	ParityEven
	ParityMark
	ParitySpace
)

// getBaudRate converts an integer baud rate to the termios constant
func getBaudRate(rate int) (uint32, error) {
	switch rate {
	case 50:
		return unix.B50, nil
	case 75:
		return unix.B75, nil
	case 110:
		return unix.B110, nil
	case 134:
		return unix.B134, nil
	case 150:
		return unix.B150, nil
	case 200:
		return unix.B200, nil
	case 300:
		return unix.B300, nil
	case 600:
		return unix.B600, nil
	case 1200:
		return unix.B1200, nil
	case 1800:
		return unix.B1800, nil
	case 2400:
		return unix.B2400, nil
	case 4800:
		return unix.B4800, nil
	case 9600:
		return unix.B9600, nil
	case 19200:
		return unix.B19200, nil
	case 38400:
		return unix.B38400, nil
	case 57600:
		return unix.B57600, nil
	case 115200:
		return unix.B115200, nil
	case 230400:
		return unix.B230400, nil
	case 460800:
		return unix.B460800, nil
	case 500000:
		return unix.B500000, nil
	case 576000:
		return unix.B576000, nil
	case 921600:
		return unix.B921600, nil
	case 1000000:
		return unix.B1000000, nil
	case 1152000:
		return unix.B1152000, nil
	case 1500000:
		return unix.B1500000, nil
	case 2000000:
		return unix.B2000000, nil
	case 2500000:
		return unix.B2500000, nil
	case 3000000:
		return unix.B3000000, nil
	case 3500000:
		return unix.B3500000, nil
	case 4000000:
		return unix.B4000000, nil
	default:
		return 0, ErrInvalidBaudRate
	}
}

// setTermios applies termios settings to a file descriptor
func setTermios(fd int, termios *unix.Termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), unix.TCSETS, uintptr(unsafe.Pointer(termios)))
	if errno != 0 {
		return errno
	}
	return nil
}

// getTermios retrieves termios settings from a file descriptor
func getTermios(fd int, termios *unix.Termios) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), unix.TCGETS, uintptr(unsafe.Pointer(termios)))
	if errno != 0 {
		return errno
	}
	return nil
}

// getModemStatus retrieves modem control signals
func getModemStatus(fd int) (int, error) {
	var status int
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), unix.TIOCMGET, uintptr(unsafe.Pointer(&status)))
	if errno != 0 {
		return 0, errno
	}
	return status, nil
}

// Open opens a serial port with the given device path and options
func Open(device string, opts ...Option) (*Port, error) {
	// Apply default configuration
	config := DefaultConfig()
	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}

	// Open the device file
	file, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrDeviceNotFound
		}
		if os.IsPermission(err) {
			return nil, ErrPermissionDenied
		}
		return nil, err
	}

	fd := int(file.Fd())

	// Save original terminal settings
	var originalTerm unix.Termios
	if err := getTermios(fd, &originalTerm); err != nil {
		file.Close()
		return nil, err
	}

	// Configure the port
	var term unix.Termios
	if err := getTermios(fd, &term); err != nil {
		file.Close()
		return nil, err
	}

	// Configure baud rate
	baudRate, err := getBaudRate(config.BaudRate)
	if err != nil {
		file.Close()
		return nil, err
	}

	// Set input and output baud rates
	term.Ispeed = baudRate
	term.Ospeed = baudRate

	// Configure control flags
	term.Cflag &^= unix.CSIZE | unix.CSTOPB | unix.PARENB | unix.PARODD | unix.CRTSCTS
	term.Cflag |= unix.CREAD | unix.CLOCAL

	// Data bits
	switch config.DataBits {
	case 5:
		term.Cflag |= unix.CS5
	case 6:
		term.Cflag |= unix.CS6
	case 7:
		term.Cflag |= unix.CS7
	case 8:
		term.Cflag |= unix.CS8
	default:
		file.Close()
		return nil, ErrInvalidConfig
	}

	// Stop bits
	if config.StopBits == 2 {
		term.Cflag |= unix.CSTOPB
	}

	// Parity
	switch config.Parity {
	case ParityNone:
		// No parity (already cleared above)
	case ParityOdd:
		term.Cflag |= unix.PARENB | unix.PARODD
	case ParityEven:
		term.Cflag |= unix.PARENB
	case ParityMark:
		term.Cflag |= unix.PARENB | unix.PARODD | unix.CMSPAR
	case ParitySpace:
		term.Cflag |= unix.PARENB | unix.CMSPAR
	default:
		file.Close()
		return nil, ErrInvalidConfig
	}

	// Flow control
	switch config.FlowControl {
	case FlowControlNone:
		// No flow control (already cleared above)
	case FlowControlCTS, FlowControlRTSCTS:
		term.Cflag |= unix.CRTSCTS
	}

	// Input flags - disable canonical mode, echo, etc.
	term.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP |
		unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON

	// Output flags - disable output processing
	term.Oflag &^= unix.OPOST

	// Local flags - disable canonical mode and echo
	term.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN

	// Control characters - blocking read (wait for at least 1 byte, no timeout)
	term.Cc[unix.VMIN] = 1
	term.Cc[unix.VTIME] = 0

	// Apply the configuration
	if err := setTermios(fd, &term); err != nil {
		file.Close()
		return nil, err
	}

	port := &Port{
		file:         file,
		config:       config,
		originalTerm: originalTerm,
		closed:       false,
	}

	return port, nil
}

// Close closes the serial port and restores original settings
func (p *Port) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPortClosed
	}

	// Restore original terminal settings
	if err := setTermios(int(p.file.Fd()), &p.originalTerm); err != nil {
		// Continue with closing even if restore fails
	}

	err := p.file.Close()
	p.closed = true
	return err
}

// Read reads data from the serial port (blocking)
func (p *Port) Read(buf []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return 0, ErrPortClosed
	}

	return p.file.Read(buf)
}

// Write writes data to the serial port (blocking)
func (p *Port) Write(data []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return 0, ErrPortClosed
	}

	// Handle flow control if needed
	if p.config.FlowControl != FlowControlNone {
		return p.writeWithFlowControl(data)
	}

	return p.file.Write(data)
}

// writeWithFlowControl handles writing with CTS flow control
func (p *Port) writeWithFlowControl(data []byte) (int, error) {
	fd := int(p.file.Fd())

	// Check CTS before writing
	status, err := getModemStatus(fd)
	if err != nil {
		return 0, err
	}

	if status&unix.TIOCM_CTS == 0 {
		// CTS is not asserted - wait or timeout
		return 0, ErrCTSTimeout
	}

	return p.file.Write(data)
}

// ReadContext reads data from the serial port with context timeout support
func (p *Port) ReadContext(ctx context.Context, buf []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return 0, ErrPortClosed
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Create a channel for the read result
	type readResult struct {
		n   int
		err error
	}
	resultCh := make(chan readResult, 1)

	// Perform the read in a goroutine
	go func() {
		n, err := p.file.Read(buf)
		resultCh <- readResult{n: n, err: err}
	}()

	// Wait for either the read to complete or context to be cancelled
	select {
	case result := <-resultCh:
		return result.n, result.err
	case <-ctx.Done():
		return 0, ErrReadTimeout
	}
}

// WriteContext writes data to the serial port with context timeout support
func (p *Port) WriteContext(ctx context.Context, data []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return 0, ErrPortClosed
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	// Handle flow control with context timeout
	if p.config.FlowControl != FlowControlNone {
		return p.writeWithFlowControlContext(ctx, data)
	}

	// Create a channel for the write result
	type writeResult struct {
		n   int
		err error
	}
	resultCh := make(chan writeResult, 1)

	// Perform the write in a goroutine
	go func() {
		n, err := p.file.Write(data)
		resultCh <- writeResult{n: n, err: err}
	}()

	// Wait for either the write to complete or context to be cancelled
	select {
	case result := <-resultCh:
		return result.n, result.err
	case <-ctx.Done():
		return 0, ErrWriteTimeout
	}
}

// writeWithFlowControlContext handles writing with CTS flow control and context timeout
func (p *Port) writeWithFlowControlContext(ctx context.Context, data []byte) (int, error) {
	fd := int(p.file.Fd())

	// Check if CTS is already asserted (fast path)
	status, err := getModemStatus(fd)
	if err != nil {
		return 0, err
	}
	if status&unix.TIOCM_CTS != 0 {
		// CTS is ready, proceed with write immediately
		return p.writeDataWithContext(ctx, data)
	}

	// CTS not ready, wait for it using interrupt-driven approach
	return p.waitForCTSAndWrite(ctx, fd, data)
}

// waitForCTSAndWrite uses epoll to efficiently wait for CTS signal changes
func (p *Port) waitForCTSAndWrite(ctx context.Context, fd int, data []byte) (int, error) {
	// Create epoll instance
	epfd, err := unix.EpollCreate1(unix.EPOLL_CLOEXEC)
	if err != nil {
		// Fallback to polling if epoll fails
		return p.waitForCTSPolling(ctx, fd, data)
	}
	defer unix.Close(epfd)

	// Add serial port fd to epoll for exceptional conditions (modem status changes)
	event := unix.EpollEvent{
		Events: unix.EPOLLPRI, // Priority/exceptional data (modem signal changes)
		Fd:     int32(fd),
	}
	if err := unix.EpollCtl(epfd, unix.EPOLL_CTL_ADD, fd, &event); err != nil {
		// Fallback to polling if epoll setup fails
		return p.waitForCTSPolling(ctx, fd, data)
	}

	// Set up timeout
	deadline := time.Now().Add(p.config.CTSTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	for time.Now().Before(deadline) {
		// Calculate remaining timeout in milliseconds
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		timeoutMs := int(remaining.Milliseconds())
		if timeoutMs <= 0 {
			timeoutMs = 1 // Minimum 1ms timeout
		}

		// Wait for events or timeout
		events := make([]unix.EpollEvent, 1)
		n, err := unix.EpollWait(epfd, events, timeoutMs)
		if err != nil {
			if err == unix.EINTR {
				continue // Interrupted system call, check context and try again
			}
			// Fallback to polling on epoll error
			return p.waitForCTSPolling(ctx, fd, data)
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return 0, ErrWriteTimeout
		default:
		}

		if n > 0 {
			// Signal change detected, check CTS status
			status, err := getModemStatus(fd)
			if err != nil {
				return 0, err
			}
			if status&unix.TIOCM_CTS != 0 {
				// CTS is now asserted, proceed with write
				return p.writeDataWithContext(ctx, data)
			}
			// CTS still not ready, continue waiting
		}
		// n == 0 means timeout, continue loop to check overall deadline
	}

	// Determine which timeout occurred
	select {
	case <-ctx.Done():
		return 0, ErrWriteTimeout
	default:
		return 0, ErrCTSTimeout
	}
}

// waitForCTSPolling is the fallback polling implementation
func (p *Port) waitForCTSPolling(ctx context.Context, fd int, data []byte) (int, error) {
	// Set up timeout
	deadline := time.Now().Add(p.config.CTSTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	// Use adaptive polling: start with 1μs, increase to 100μs max
	pollInterval := time.Microsecond
	const maxPollInterval = 100 * time.Microsecond
	const intervalGrowthFactor = 2

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return 0, ErrWriteTimeout
		default:
		}

		status, err := getModemStatus(fd)
		if err != nil {
			return 0, err
		}

		if status&unix.TIOCM_CTS != 0 {
			// CTS is asserted, proceed with write
			return p.writeDataWithContext(ctx, data)
		}

		// Sleep with adaptive interval
		time.Sleep(pollInterval)
		if pollInterval < maxPollInterval {
			pollInterval *= intervalGrowthFactor
		}
	}

	// Determine which timeout occurred
	select {
	case <-ctx.Done():
		return 0, ErrWriteTimeout
	default:
		return 0, ErrCTSTimeout
	}
}

// writeDataWithContext performs the actual write with context cancellation support
func (p *Port) writeDataWithContext(ctx context.Context, data []byte) (int, error) {
	type writeResult struct {
		n   int
		err error
	}
	resultCh := make(chan writeResult, 1)

	// Perform the write in a goroutine
	go func() {
		n, err := p.file.Write(data)
		resultCh <- writeResult{n: n, err: err}
	}()

	// Wait for either the write to complete or context to be cancelled
	select {
	case result := <-resultCh:
		return result.n, result.err
	case <-ctx.Done():
		return 0, ErrWriteTimeout
	}
}
