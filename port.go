package serial

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

// Port represents a serial port connection interface
type Port interface {
	Close() error
	Read(buf []byte) (int, error)
	Write(data []byte) (int, error)
	WriteContext(ctx context.Context, data []byte) (int, error)
	ReadContext(ctx context.Context, buf []byte) (int, error)
	GetCTSStatus() (bool, error)
	Drain() error
	FlushInput() error
	FlushOutput() error

	// Modem signal control and monitoring
	GetModemSignals() (ModemSignals, error)
	SetRTS(state bool) error
	GetRTS() (bool, error)
	WaitForSignalChange(mask SignalMask, timeout time.Duration) (ModemSignals, SignalMask, error)
	WaitForSignalChangeContext(ctx context.Context, mask SignalMask) (ModemSignals, SignalMask, error)
}

// port is the concrete implementation of the Port interface
type port struct {
	mu         sync.RWMutex
	fd         int
	config     Config
	closed     bool
	ctsMonitor *ctsMonitor // CTS monitoring for flow control
}

// Ensure port implements Port interface at compile time
var _ Port = (*port)(nil)

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

// ModemSignals represents modem control signal states
type ModemSignals struct {
	CTS bool // Clear To Send
	DSR bool // Data Set Ready
	RI  bool // Ring Indicator
	DCD bool // Data Carrier Detect
	RTS bool // Request To Send
	DTR bool // Data Terminal Ready
}

// SignalMask identifies which signals to monitor
type SignalMask int

const (
	SignalCTS SignalMask = 1 << iota
	SignalDSR
	SignalRI
	SignalDCD
)

// ctsMonitor handles CTS signal monitoring using TIOCMIWAIT
type ctsMonitor struct {
	fd       int
	stopCh   chan struct{}
	activeCh chan struct{}
}

// getBaudRate converts an integer baud rate to the unix constant
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

// getModemStatus retrieves modem control signals using unix package
func getModemStatus(fd int) (int, error) {
	return unix.IoctlGetInt(fd, unix.TIOCMGET)
}

// assertRTS manually asserts the RTS signal using unix package
func assertRTS(fd int) error {
	return unix.IoctlSetInt(fd, unix.TIOCMBIS, unix.TIOCM_RTS)
}

// setDTR sets DTR signal state
func setDTR(fd int, state bool) error {
	if state {
		return unix.IoctlSetInt(fd, unix.TIOCMBIS, unix.TIOCM_DTR)
	}
	return unix.IoctlSetInt(fd, unix.TIOCMBIC, unix.TIOCM_DTR)
}

// setRTSSignal sets RTS signal state
func setRTSSignal(fd int, state bool) error {
	if state {
		return unix.IoctlSetInt(fd, unix.TIOCMBIS, unix.TIOCM_RTS)
	}
	return unix.IoctlSetInt(fd, unix.TIOCMBIC, unix.TIOCM_RTS)
}

// waitForCTSChange waits for CTS signal changes using TIOCMIWAIT
func waitForCTSChange(fd int) error {
	return unix.IoctlSetInt(fd, unix.TIOCMIWAIT, unix.TIOCM_CTS)
}

// signalMaskToTIOCM converts SignalMask to unix TIOCM bits
func signalMaskToTIOCM(mask SignalMask) int {
	var bits int
	if mask&SignalCTS != 0 {
		bits |= unix.TIOCM_CTS
	}
	if mask&SignalDSR != 0 {
		bits |= unix.TIOCM_DSR
	}
	if mask&SignalRI != 0 {
		bits |= unix.TIOCM_RI
	}
	if mask&SignalDCD != 0 {
		bits |= unix.TIOCM_CAR
	}
	return bits
}

// detectSignalChanges compares old and new signal states to determine what changed
func detectSignalChanges(oldStatus, newStatus int) SignalMask {
	var changed SignalMask
	if (oldStatus&unix.TIOCM_CTS != 0) != (newStatus&unix.TIOCM_CTS != 0) {
		changed |= SignalCTS
	}
	if (oldStatus&unix.TIOCM_DSR != 0) != (newStatus&unix.TIOCM_DSR != 0) {
		changed |= SignalDSR
	}
	if (oldStatus&unix.TIOCM_RI != 0) != (newStatus&unix.TIOCM_RI != 0) {
		changed |= SignalRI
	}
	if (oldStatus&unix.TIOCM_CAR != 0) != (newStatus&unix.TIOCM_CAR != 0) {
		changed |= SignalDCD
	}
	return changed
}

// newCTSMonitor creates a new CTS monitor
func newCTSMonitor(fd int) *ctsMonitor {
	return &ctsMonitor{
		fd:       fd,
		stopCh:   make(chan struct{}),
		activeCh: make(chan struct{}, 1), // Buffered to prevent blocking
	}
}

// start begins CTS monitoring in a background goroutine
func (c *ctsMonitor) start() {
	go func() {
		for {
			select {
			case <-c.stopCh:
				return
			default:
				// Wait for CTS signal change
				err := waitForCTSChange(c.fd)
				if err != nil {
					// Error occurred, exit monitoring
					return
				}

				// Check if CTS is now active
				status, err := getModemStatus(c.fd)
				if err != nil {
					continue
				}

				if status&unix.TIOCM_CTS != 0 {
					// CTS is active, signal waiting write operations
					select {
					case c.activeCh <- struct{}{}:
					default:
						// Channel already has a signal, skip
					}
				}
			}
		}
	}()
}

// stop stops CTS monitoring
func (c *ctsMonitor) stop() {
	close(c.stopCh)
}

// waitForCTS waits for CTS to become active with timeout
func (c *ctsMonitor) waitForCTS(timeout time.Duration) error {
	// First check if CTS is already active
	status, err := getModemStatus(c.fd)
	if err != nil {
		return err
	}
	if status&unix.TIOCM_CTS != 0 {
		return nil // CTS already active
	}

	// Wait for CTS to become active
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-c.activeCh:
		return nil
	case <-timer.C:
		return ErrCTSTimeout
	case <-c.stopCh:
		return ErrPortClosed
	}
}

// Open opens a serial port with the given device path and options
func Open(device string, opts ...Option) (Port, error) {
	// Apply default configuration
	config := DefaultConfig()
	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return nil, err
		}
	}

	// Open device file using unix.Open for better control
	flags := unix.O_RDWR | unix.O_NOCTTY
	if config.WriteMode == WriteModeSynced {
		flags |= unix.O_SYNC
	}

	fd, err := unix.Open(device, flags, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %v", device, err)
	}

	// Configure port with simple termios setup
	if err := configurePort(fd, config); err != nil {
		unix.Close(fd)
		return nil, err
	}

	// Apply initial signal states if configured
	if config.InitialRTS != nil {
		if err := setRTSSignal(fd, *config.InitialRTS); err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("failed to set initial RTS: %v", err)
		}
	}
	if config.InitialDTR != nil {
		if err := setDTR(fd, *config.InitialDTR); err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("failed to set initial DTR: %v", err)
		}
	}

	p := &port{
		fd:     fd,
		config: config,
		closed: false,
	}

	// Set up CTS monitoring if flow control is enabled
	if config.FlowControl == FlowControlCTS {
		p.ctsMonitor = newCTSMonitor(fd)
		p.ctsMonitor.start()
	}

	return p, nil
}

// configurePort configures the serial port using clean unix package calls
func configurePort(fd int, config Config) error {
	// Get current termios settings
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return fmt.Errorf("failed to get termios: %v", err)
	}

	// Configure for raw mode, 8N1 by default
	termios.Cflag = unix.CS8 | unix.CREAD | unix.CLOCAL
	termios.Iflag = 0 // No input processing
	termios.Oflag = 0 // No output processing
	termios.Lflag = 0 // No line processing (raw mode)

	// Timeout: VMIN=0, VTIME from config (deciseconds)
	termios.Cc[unix.VMIN] = 0
	termios.Cc[unix.VTIME] = uint8(config.ReadTimeoutTenths)

	// Get and set baud rate
	baudRate, err := getBaudRate(config.BaudRate)
	if err != nil {
		return err
	}

	// Set speed directly in termios structure
	termios.Cflag = (termios.Cflag &^ unix.CBAUD) | baudRate
	termios.Ispeed = baudRate
	termios.Ospeed = baudRate

	// Apply config-specific settings
	// Data bits
	if config.DataBits != 8 {
		termios.Cflag &^= unix.CSIZE
		switch config.DataBits {
		case 5:
			termios.Cflag |= unix.CS5
		case 6:
			termios.Cflag |= unix.CS6
		case 7:
			termios.Cflag |= unix.CS7
		case 8:
			termios.Cflag |= unix.CS8
		}
	}

	// Stop bits
	if config.StopBits == 2 {
		termios.Cflag |= unix.CSTOPB
	}

	// Parity
	switch config.Parity {
	case ParityOdd:
		termios.Cflag |= unix.PARENB | unix.PARODD
	case ParityEven:
		termios.Cflag |= unix.PARENB
	}

	// Flow control
	if config.FlowControl == FlowControlRTSCTS {
		termios.Cflag |= unix.CRTSCTS
	}

	// Apply settings immediately
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, termios); err != nil {
		return fmt.Errorf("failed to set termios: %v", err)
	}

	// For RTS/CTS flow control, ensure RTS is asserted to signal readiness
	if config.FlowControl == FlowControlRTSCTS {
		if err := assertRTS(fd); err != nil {
			// Non-fatal - some systems might not support manual RTS control
		}
	}

	return nil
}

// Close closes the serial port
func (p *port) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPortClosed
	}

	// Stop CTS monitoring if active
	if p.ctsMonitor != nil {
		p.ctsMonitor.stop()
	}

	err := unix.Close(p.fd)
	p.closed = true
	return err
}

// Read reads data from the serial port
func (p *port) Read(buf []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return 0, ErrPortClosed
	}

	return unix.Read(p.fd, buf)
}

// Write writes data to the serial port
func (p *port) Write(data []byte) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return 0, ErrPortClosed
	}

	// Handle CTS flow control if enabled
	if p.config.FlowControl == FlowControlCTS && p.ctsMonitor != nil {
		if err := p.ctsMonitor.waitForCTS(p.config.CTSTimeout); err != nil {
			return 0, err
		}
	}

	// Perform the write
	return unix.Write(p.fd, data)
}

// WriteContext writes data with context timeout support
func (p *port) WriteContext(ctx context.Context, data []byte) (int, error) {
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

	// Handle CTS flow control with context timeout
	if p.config.FlowControl == FlowControlCTS && p.ctsMonitor != nil {
		// Use shorter of context timeout or CTS timeout
		timeout := p.config.CTSTimeout
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining < timeout {
				timeout = remaining
			}
		}

		if err := p.ctsMonitor.waitForCTS(timeout); err != nil {
			return 0, err
		}
	}

	// Create channel for write result
	type writeResult struct {
		n   int
		err error
	}
	resultCh := make(chan writeResult, 1)

	// Perform write in goroutine
	go func() {
		n, err := unix.Write(p.fd, data)
		resultCh <- writeResult{n: n, err: err}
	}()

	// Wait for write completion or context cancellation
	select {
	case result := <-resultCh:
		return result.n, result.err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// ReadContext reads data with context timeout support
func (p *port) ReadContext(ctx context.Context, buf []byte) (int, error) {
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

	// Create channel for read result
	type readResult struct {
		n   int
		err error
	}
	resultCh := make(chan readResult, 1)

	// Perform read in goroutine
	go func() {
		n, err := unix.Read(p.fd, buf)
		resultCh <- readResult{n: n, err: err}
	}()

	// Wait for read completion or context cancellation
	select {
	case result := <-resultCh:
		return result.n, result.err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// GetCTSStatus returns the current CTS status
func (p *port) GetCTSStatus() (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return false, ErrPortClosed
	}

	status, err := getModemStatus(p.fd)
	if err != nil {
		return false, err
	}

	return status&unix.TIOCM_CTS != 0, nil
}

// GetModemSignals returns current state of all modem control signals
func (p *port) GetModemSignals() (ModemSignals, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ModemSignals{}, ErrPortClosed
	}

	status, err := getModemStatus(p.fd)
	if err != nil {
		return ModemSignals{}, err
	}

	return ModemSignals{
		CTS: status&unix.TIOCM_CTS != 0,
		DSR: status&unix.TIOCM_DSR != 0,
		RI:  status&unix.TIOCM_RI != 0,
		DCD: status&unix.TIOCM_CAR != 0,
		RTS: status&unix.TIOCM_RTS != 0,
		DTR: status&unix.TIOCM_DTR != 0,
	}, nil
}

// SetRTS manually sets the RTS signal state
// When true, asserts RTS (signals readiness to receive)
// When false, deasserts RTS (signals not ready)
func (p *port) SetRTS(state bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPortClosed
	}

	if state {
		return unix.IoctlSetInt(p.fd, unix.TIOCMBIS, unix.TIOCM_RTS)
	}
	return unix.IoctlSetInt(p.fd, unix.TIOCMBIC, unix.TIOCM_RTS)
}

// GetRTS returns current RTS signal state
func (p *port) GetRTS() (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return false, ErrPortClosed
	}

	status, err := getModemStatus(p.fd)
	if err != nil {
		return false, err
	}

	return status&unix.TIOCM_RTS != 0, nil
}

// WaitForSignalChange blocks until any monitored signal changes state
// Returns new signal states and which signal(s) changed
func (p *port) WaitForSignalChange(mask SignalMask, timeout time.Duration) (ModemSignals, SignalMask, error) {
	if mask == 0 {
		return ModemSignals{}, 0, ErrInvalidSignalMask
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ModemSignals{}, 0, ErrPortClosed
	}
	fd := p.fd
	p.mu.RUnlock()

	// Get initial signal state
	oldStatus, err := getModemStatus(fd)
	if err != nil {
		return ModemSignals{}, 0, err
	}

	// Convert mask to TIOCM bits
	tiocmBits := signalMaskToTIOCM(mask)

	// Channel for wait result
	type waitResult struct {
		newStatus int
		err       error
	}
	resultCh := make(chan waitResult, 1)

	// Wait for signal change in goroutine
	go func() {
		err := unix.IoctlSetInt(fd, unix.TIOCMIWAIT, tiocmBits)
		if err != nil {
			resultCh <- waitResult{err: err}
			return
		}

		// Get new status after change
		newStatus, err := getModemStatus(fd)
		resultCh <- waitResult{newStatus: newStatus, err: err}
	}()

	// Wait for result or timeout
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-resultCh:
		if result.err != nil {
			return ModemSignals{}, 0, result.err
		}

		// Detect which signals changed
		changed := detectSignalChanges(oldStatus, result.newStatus)

		// Convert to ModemSignals
		signals := ModemSignals{
			CTS: result.newStatus&unix.TIOCM_CTS != 0,
			DSR: result.newStatus&unix.TIOCM_DSR != 0,
			RI:  result.newStatus&unix.TIOCM_RI != 0,
			DCD: result.newStatus&unix.TIOCM_CAR != 0,
			RTS: result.newStatus&unix.TIOCM_RTS != 0,
			DTR: result.newStatus&unix.TIOCM_DTR != 0,
		}

		return signals, changed, nil

	case <-timer.C:
		return ModemSignals{}, 0, ErrSignalTimeout
	}
}

// WaitForSignalChangeContext waits with context cancellation support
func (p *port) WaitForSignalChangeContext(ctx context.Context, mask SignalMask) (ModemSignals, SignalMask, error) {
	if mask == 0 {
		return ModemSignals{}, 0, ErrInvalidSignalMask
	}

	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ModemSignals{}, 0, ErrPortClosed
	}
	fd := p.fd
	p.mu.RUnlock()

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		return ModemSignals{}, 0, ctx.Err()
	default:
	}

	// Get initial signal state
	oldStatus, err := getModemStatus(fd)
	if err != nil {
		return ModemSignals{}, 0, err
	}

	// Convert mask to TIOCM bits
	tiocmBits := signalMaskToTIOCM(mask)

	// Channel for wait result
	type waitResult struct {
		newStatus int
		err       error
	}
	resultCh := make(chan waitResult, 1)

	// Wait for signal change in goroutine
	go func() {
		err := unix.IoctlSetInt(fd, unix.TIOCMIWAIT, tiocmBits)
		if err != nil {
			resultCh <- waitResult{err: err}
			return
		}

		// Get new status after change
		newStatus, err := getModemStatus(fd)
		resultCh <- waitResult{newStatus: newStatus, err: err}
	}()

	// Wait for result or context cancellation
	select {
	case result := <-resultCh:
		if result.err != nil {
			return ModemSignals{}, 0, result.err
		}

		// Detect which signals changed
		changed := detectSignalChanges(oldStatus, result.newStatus)

		// Convert to ModemSignals
		signals := ModemSignals{
			CTS: result.newStatus&unix.TIOCM_CTS != 0,
			DSR: result.newStatus&unix.TIOCM_DSR != 0,
			RI:  result.newStatus&unix.TIOCM_RI != 0,
			DCD: result.newStatus&unix.TIOCM_CAR != 0,
			RTS: result.newStatus&unix.TIOCM_RTS != 0,
			DTR: result.newStatus&unix.TIOCM_DTR != 0,
		}

		return signals, changed, nil

	case <-ctx.Done():
		return ModemSignals{}, 0, ctx.Err()
	}
}

// Drain waits until all output written to the port has been transmitted
func (p *port) Drain() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPortClosed
	}

	return unix.IoctlSetInt(p.fd, unix.TCSBRK, 1)
}

// FlushInput discards any unread input data
func (p *port) FlushInput() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPortClosed
	}

	return unix.IoctlSetInt(p.fd, unix.TCFLSH, unix.TCIFLUSH)
}

// FlushOutput discards any unwritten output data
func (p *port) FlushOutput() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPortClosed
	}

	return unix.IoctlSetInt(p.fd, unix.TCFLSH, unix.TCOFLUSH)
}
