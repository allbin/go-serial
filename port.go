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
	DrainOutput() error
	DrainInput() error
	FlushInput() error
	FlushOutput() error

	// Modem signal control and monitoring
	GetModemSignals() (ModemSignals, error)
	SetRTS(state bool) error
	GetRTS() (bool, error)
	SetDTR(state bool) error
	GetDTR() (bool, error)
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

// writeRequest represents a queued write operation waiting for CTS
type writeRequest struct {
	data     []byte
	timeout  time.Duration
	cancelCh <-chan struct{} // closed by the requester on timeout or context cancellation
	resultCh chan writeResult
}

// writeResult contains the result of a write operation
type writeResult struct {
	n   int
	err error
}

// ctsMonitor paces writes for event-based CTS devices (e.g. NeoCortec NeoMesh).
//
// Such devices assert CTS for a sub-millisecond window once per scheduled
// event and only accept data that arrives inside that window. USB-serial
// adapters deliver CTS edges to userspace too late to write into the current
// window, so the actual byte-level gating is delegated to the adapter chip
// via termios CRTSCTS (see configurePort). The monitor's job is pacing:
//
//  1. Wait until CTS is inactive, so the frame is never handed to the chip
//     while a window is already open (a late start can split the frame
//     across two windows and the device drops it).
//  2. Write the frame. CRTSCTS holds it in the chip FIFO and transmits it
//     contiguously the moment the next window opens.
//  3. Drain, so completion means "frame physically transmitted" and at most
//     one frame is in flight per CTS window.
//
// This mirrors the vendor reference implementation (NeoTools), which enables
// hardware handshake on every platform and queues frames on the CTS
// deassert edge.
type ctsMonitor struct {
	fd      int
	stopCh  chan struct{}
	writeCh chan *writeRequest // Queue for pending writes
	edgeCh  chan struct{}      // CTS edge notifications from the watcher goroutine
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

// getModemStatus retrieves modem control signals using unix package.
// Retries on EINTR: blocking tty ioctls are routinely interrupted by the Go
// runtime's preemption signals.
func getModemStatus(fd int) (int, error) {
	for {
		status, err := unix.IoctlGetInt(fd, unix.TIOCMGET)
		if err != unix.EINTR {
			return status, err
		}
	}
}

// setDTR sets DTR signal state
func setDTR(fd int, state bool) error {
	// Read current modem status
	status, err := unix.IoctlGetInt(fd, unix.TIOCMGET)
	if err != nil {
		return err
	}

	// Modify DTR bit
	if state {
		status |= unix.TIOCM_DTR
	} else {
		status &^= unix.TIOCM_DTR
	}

	// Write back using TIOCMSET
	return unix.IoctlSetPointerInt(fd, unix.TIOCMSET, status)
}

// setRTSSignal sets RTS signal state
func setRTSSignal(fd int, state bool) error {
	// Read current modem status
	status, err := unix.IoctlGetInt(fd, unix.TIOCMGET)
	if err != nil {
		return err
	}

	// Modify RTS bit
	if state {
		status |= unix.TIOCM_RTS
	} else {
		status &^= unix.TIOCM_RTS
	}

	// Write back using TIOCMSET
	return unix.IoctlSetPointerInt(fd, unix.TIOCMSET, status)
}

// waitForCTSChange waits for CTS signal changes using TIOCMIWAIT,
// retrying when the wait is interrupted by a signal
func waitForCTSChange(fd int) error {
	for {
		err := unix.IoctlSetInt(fd, unix.TIOCMIWAIT, unix.TIOCM_CTS)
		if err != unix.EINTR {
			return err
		}
	}
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
		fd:      fd,
		stopCh:  make(chan struct{}),
		writeCh: make(chan *writeRequest, 1), // Buffered for one pending write
		edgeCh:  make(chan struct{}, 1),
	}
}

// start begins CTS monitoring in background goroutines
func (c *ctsMonitor) start() {
	// Single long-lived edge watcher. TIOCMIWAIT has no timeout, so at most
	// this one goroutine can remain parked in the ioctl after the port
	// closes, instead of one per attempted write.
	go func() {
		for {
			if err := waitForCTSChange(c.fd); err != nil {
				return
			}
			select {
			case <-c.stopCh:
				return
			case c.edgeCh <- struct{}{}:
			default: // an edge is already pending, coalesce
			}
		}
	}()

	go func() {
		for {
			select {
			case <-c.stopCh:
				return
			case req := <-c.writeCh:
				req.resultCh <- c.execute(req)
			}
		}
	}()
}

// execute performs one paced write: wait for CTS inactive, arm the frame,
// wait for it to be transmitted.
func (c *ctsMonitor) execute(req *writeRequest) writeResult {
	// Requester may have given up while the request sat in the queue
	select {
	case <-req.cancelCh:
		return writeResult{0, ErrCTSTimeout}
	default:
	}

	timer := time.NewTimer(req.timeout)
	defer timer.Stop()

	// Discard any stale edge from a previous CTS pulse
	select {
	case <-c.edgeCh:
	default:
	}

	// Phase 1: wait until CTS is inactive so the frame arms for the NEXT
	// window rather than starting mid-window. On a NeoCortec module this
	// waits at most one CTS pulse width (~488us). If CTS is held active
	// continuously (bootloader mode, classic flow-control devices), the
	// timer fires and we write anyway - the chip transmits immediately in
	// that case, which is the correct behavior for a continuously-ready
	// device. This matches the vendor reference (write on CTS deassert
	// edge; on timeout write only if CTS is still asserted).
waitInactive:
	for {
		status, err := getModemStatus(c.fd)
		if err != nil {
			return writeResult{0, err}
		}
		if status&unix.TIOCM_CTS == 0 {
			break
		}
		select {
		case <-c.edgeCh:
			// CTS changed, re-check level
		case <-timer.C:
			break waitInactive
		case <-req.cancelCh:
			return writeResult{0, ErrCTSTimeout}
		case <-c.stopCh:
			return writeResult{0, ErrPortClosed}
		}
	}

	// Phase 2: arm the frame. With CRTSCTS enabled the adapter chip holds
	// it and transmits contiguously from the first microseconds of the next
	// CTS window - timing precision no host-side code can achieve.
	n, err := unix.Write(c.fd, req.data)
	if err != nil {
		return writeResult{0, err}
	}

	// Phase 3: wait until the frame has physically left the chip (the next
	// CTS window). This paces callers to one frame per window and makes a
	// returned write mean "transmitted", not "buffered". Bounded by its own
	// timeout so a dead module cannot leave a stale frame armed in the chip
	// FIFO: on timeout/cancel the output buffer is flushed.
	drainTimer := time.NewTimer(req.timeout)
	defer drainTimer.Stop()
	drained := make(chan error, 1)
	go func() {
		for {
			err := unix.IoctlSetInt(c.fd, unix.TCSBRK, 1)
			if err != unix.EINTR {
				drained <- err
				return
			}
		}
	}()

	select {
	case err := <-drained:
		if err != nil {
			return writeResult{n, err}
		}
		return writeResult{n, nil}
	case <-drainTimer.C:
		unix.IoctlSetInt(c.fd, unix.TCFLSH, unix.TCOFLUSH)
		return writeResult{0, ErrCTSTimeout}
	case <-req.cancelCh:
		unix.IoctlSetInt(c.fd, unix.TCFLSH, unix.TCOFLUSH)
		return writeResult{0, ErrCTSTimeout}
	case <-c.stopCh:
		return writeResult{0, ErrPortClosed}
	}
}

// stop stops CTS monitoring
func (c *ctsMonitor) stop() {
	close(c.stopCh)
}

// queueWrite queues a write operation and waits for it to complete.
// Completion means the frame has been transmitted on the wire (inside a CTS
// window), not merely buffered. cancel may be nil; if it fires, any armed
// but untransmitted data is flushed so it cannot leak into a later window.
func (c *ctsMonitor) queueWrite(data []byte, timeout time.Duration, cancel <-chan struct{}) (int, error) {
	req := &writeRequest{
		data:     data,
		timeout:  timeout,
		cancelCh: cancel,
		resultCh: make(chan writeResult, 1),
	}

	// Try to enqueue the write request
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case c.writeCh <- req:
		// Request queued successfully, wait for result
	case <-timer.C:
		return 0, ErrCTSTimeout
	case <-cancel:
		return 0, ErrCTSTimeout
	case <-c.stopCh:
		return 0, ErrPortClosed
	}

	// Wait for the result. The monitor owns the request from here: it
	// observes the same cancel channel and cleans up armed data itself.
	select {
	case result := <-req.resultCh:
		return result.n, result.err
	case <-c.stopCh:
		return 0, ErrPortClosed
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

	// Validate flow control configuration
	if config.FlowControl == FlowControlCTS && config.InitialRTS == nil {
		return nil, fmt.Errorf("CTS flow control requires WithInitialRTS(true) to assert RTS")
	}
	if config.FlowControl == FlowControlRTSCTS && config.InitialRTS == nil {
		return nil, fmt.Errorf("RTS/CTS flow control requires WithInitialRTS(true) to assert RTS")
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
		// Verify RTS was set
		status, err := getModemStatus(fd)
		if err != nil {
			unix.Close(fd)
			return nil, fmt.Errorf("failed to verify initial RTS: %v", err)
		}
		rtsState := status&unix.TIOCM_RTS != 0
		if rtsState != *config.InitialRTS {
			unix.Close(fd)
			return nil, fmt.Errorf("initial RTS verification failed: requested %v, got %v", *config.InitialRTS, rtsState)
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
	termios.Cc[unix.VTIME] = uint8(config.ReadTimeout / (100 * time.Millisecond))

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

	// Flow control. FlowControlCTS also enables CRTSCTS: the adapter chip
	// gates TX on CTS in hardware, which is the only way to hit
	// sub-millisecond CTS windows (USB modem-status reporting is far too
	// slow for userspace gating). The ctsMonitor then only paces frames,
	// it does not gate bytes.
	if config.FlowControl == FlowControlRTSCTS || config.FlowControl == FlowControlCTS {
		termios.Cflag |= unix.CRTSCTS
	}

	// Apply settings immediately
	if err := unix.IoctlSetTermios(fd, unix.TCSETS, termios); err != nil {
		return fmt.Errorf("failed to set termios: %v", err)
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

	if p.closed {
		p.mu.RUnlock()
		return 0, ErrPortClosed
	}

	// Handle CTS flow control if enabled. The write returns once the frame
	// has been transmitted inside a CTS window (see ctsMonitor). The port
	// lock is released first: a paced write can block for seconds and must
	// not hold up Close(), which interrupts the write via the monitor's
	// stop channel instead.
	if p.config.FlowControl == FlowControlCTS && p.ctsMonitor != nil {
		monitor, timeout := p.ctsMonitor, p.config.CTSTimeout
		p.mu.RUnlock()
		return monitor.queueWrite(data, timeout, nil)
	}
	defer p.mu.RUnlock()

	// No flow control, perform direct write
	return unix.Write(p.fd, data)
}

// WriteContext writes data with context timeout support
func (p *port) WriteContext(ctx context.Context, data []byte) (int, error) {
	p.mu.RLock()

	if p.closed {
		p.mu.RUnlock()
		return 0, ErrPortClosed
	}

	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		p.mu.RUnlock()
		return 0, ctx.Err()
	default:
	}

	// Handle CTS flow control with context timeout. The context's done
	// channel is passed to the monitor so cancellation also flushes any
	// armed-but-untransmitted frame instead of letting it transmit in a
	// later CTS window. As in Write, the port lock is not held while the
	// paced write blocks.
	if p.config.FlowControl == FlowControlCTS && p.ctsMonitor != nil {
		// Use shorter of context timeout or CTS timeout
		timeout := p.config.CTSTimeout
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining < timeout {
				timeout = remaining
			}
		}

		monitor := p.ctsMonitor
		p.mu.RUnlock()
		n, err := monitor.queueWrite(data, timeout, ctx.Done())
		if err != nil && ctx.Err() != nil {
			return 0, ctx.Err()
		}
		return n, err
	}

	// No flow control, perform direct write with context
	defer p.mu.RUnlock()
	type directWriteResult struct {
		n   int
		err error
	}
	resultCh := make(chan directWriteResult, 1)

	go func() {
		n, err := unix.Write(p.fd, data)
		resultCh <- directWriteResult{n: n, err: err}
	}()

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

	// Read current modem status
	status, err := unix.IoctlGetInt(p.fd, unix.TIOCMGET)
	if err != nil {
		return err
	}

	// Modify RTS bit
	if state {
		status |= unix.TIOCM_RTS
	} else {
		status &^= unix.TIOCM_RTS
	}

	// Write back
	return unix.IoctlSetPointerInt(p.fd, unix.TIOCMSET, status)
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

// SetDTR manually sets the DTR signal state
// When true, asserts DTR (signals terminal ready)
// When false, deasserts DTR (signals terminal not ready)
func (p *port) SetDTR(state bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPortClosed
	}

	// Read current modem status
	status, err := unix.IoctlGetInt(p.fd, unix.TIOCMGET)
	if err != nil {
		return err
	}

	// Modify DTR bit
	if state {
		status |= unix.TIOCM_DTR
	} else {
		status &^= unix.TIOCM_DTR
	}

	// Write back
	return unix.IoctlSetPointerInt(p.fd, unix.TIOCMSET, status)
}

// GetDTR returns current DTR signal state
func (p *port) GetDTR() (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return false, ErrPortClosed
	}

	status, err := getModemStatus(p.fd)
	if err != nil {
		return false, err
	}

	return status&unix.TIOCM_DTR != 0, nil
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

// DrainOutput waits until all output written to the port has been transmitted
func (p *port) DrainOutput() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPortClosed
	}

	return unix.IoctlSetInt(p.fd, unix.TCSBRK, 1)
}

// FlushInput discards any unread input data in the kernel buffer
func (p *port) FlushInput() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPortClosed
	}

	return unix.IoctlSetInt(p.fd, unix.TCFLSH, unix.TCIFLUSH)
}

// DrainInput reads and discards all pending input data until the buffer is empty.
// It first flushes the kernel buffer, then actively reads until no more data arrives,
// ensuring data in transit or hardware FIFOs is also cleared.
func (p *port) DrainInput() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return ErrPortClosed
	}

	// Flush kernel buffer first
	if err := unix.IoctlSetInt(p.fd, unix.TCFLSH, unix.TCIFLUSH); err != nil {
		return err
	}

	// Read until no more data arrives
	buf := make([]byte, 256)
	for {
		n, err := unix.Read(p.fd, buf)
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
	}
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
