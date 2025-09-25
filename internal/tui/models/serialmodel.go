package models

import (
	"context"
	"sync"

	"github.com/mdjarv/serial"
	"github.com/mdjarv/serial/internal/tui/components"
)

// InputMode represents the current input mode (vim-like)
type InputMode int

const (
	InputModeNormal InputMode = iota
	InputModeInsert
)

func (m InputMode) String() string {
	switch m {
	case InputModeNormal:
		return "NORMAL"
	case InputModeInsert:
		return "INSERT"
	default:
		return "NORMAL"
	}
}

type ConnectionStatusMsg struct {
	Connected bool
	Error     error
}

type SerialModel struct {
	// Serial connection
	port     *serial.Port
	portPath string

	// State
	connected bool
	rawData   []components.DataReceivedMsg
	err       error
	ready     bool

	// Input mode (vim-like)
	inputMode InputMode

	// Data formatting
	formatter *components.DataFormatter

	// Cancellation and synchronization
	cancel context.CancelFunc
	ctx    context.Context
	mu     sync.RWMutex
}

func NewSerialModel(portPath string) *SerialModel {
	ctx, cancel := context.WithCancel(context.Background())

	return &SerialModel{
		portPath:  portPath,
		rawData:   make([]components.DataReceivedMsg, 0),
		inputMode: InputModeNormal,                         // Start in normal mode
		formatter: components.NewDataFormatter(true, true), // Default: show both hex and ASCII
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (m *SerialModel) GetPort() *serial.Port {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.port
}

func (m *SerialModel) SetPort(port *serial.Port) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.port = port
}

func (m *SerialModel) GetPortPath() string {
	return m.portPath
}

func (m *SerialModel) IsConnected() bool {
	return m.connected
}

func (m *SerialModel) SetConnected(connected bool) {
	m.connected = connected
}

func (m *SerialModel) GetError() error {
	return m.err
}

func (m *SerialModel) SetError(err error) {
	m.err = err
}

func (m *SerialModel) IsReady() bool {
	return m.ready
}

func (m *SerialModel) SetReady(ready bool) {
	m.ready = ready
}

func (m *SerialModel) GetRawData() []components.DataReceivedMsg {
	return m.rawData
}

func (m *SerialModel) AddRawData(msg components.DataReceivedMsg) {
	m.rawData = append(m.rawData, msg)
}

func (m *SerialModel) ClearData() {
	m.rawData = make([]components.DataReceivedMsg, 0)
}

func (m *SerialModel) GetFormattedData() []string {
	return m.formatter.FormatMessages(m.rawData)
}

func (m *SerialModel) FormatMessage(msg components.DataReceivedMsg) string {
	return m.formatter.FormatMessage(msg)
}

func (m *SerialModel) GetFormatter() *components.DataFormatter {
	return m.formatter
}

func (m *SerialModel) GetInputMode() InputMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.inputMode
}

func (m *SerialModel) SetInputMode(mode InputMode) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inputMode = mode
}

func (m *SerialModel) ToggleInputMode() InputMode {
	m.mu.Lock()
	defer m.mu.Unlock()
	switch m.inputMode {
	case InputModeNormal:
		m.inputMode = InputModeInsert
	case InputModeInsert:
		m.inputMode = InputModeNormal
	}
	return m.inputMode
}

func (m *SerialModel) IsInInsertMode() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.inputMode == InputModeInsert
}

func (m *SerialModel) GetContext() context.Context {
	return m.ctx
}

func (m *SerialModel) Cancel() {
	if m.cancel != nil {
		m.cancel()
	}
}

func (m *SerialModel) Cleanup() {
	// Cancel context to stop goroutines
	if m.cancel != nil {
		m.cancel()
	}

	// Close port safely
	m.mu.Lock()
	if m.port != nil {
		m.port.Close()
		m.port = nil
	}
	m.mu.Unlock()
}
