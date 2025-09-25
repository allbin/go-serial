/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mdjarv/serial"
	"github.com/spf13/cobra"
)

// listenCmd represents the listen command
var listenCmd = &cobra.Command{
	Use:   "listen <port>",
	Short: "Listen for data on a serial port with real-time display",
	Long: `Listen for incoming data on a serial port with a beautiful real-time TUI display.

This command opens the specified serial port and displays incoming data in real-time
using a terminal user interface. Features include:
- Real-time data streaming with timestamps
- ASCII and hex display modes
- Connection status indicators
- Configurable baud rate and flow control
- Clean, responsive interface

Example usage:
  serial listen /dev/ttyUSB0
  serial listen /dev/ttyUSB0 --baud 9600
  serial listen /dev/ttyUSB0 --flow-control cts`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		portPath := args[0]

		// Get flags
		baudRate, _ := cmd.Flags().GetInt("baud")
		flowControl, _ := cmd.Flags().GetString("flow-control")

		// Configure port options
		opts := []serial.Option{
			serial.WithBaudRate(baudRate),
		}

		switch strings.ToLower(flowControl) {
		case "cts":
			opts = append(opts, serial.WithFlowControl(serial.FlowControlCTS))
		case "rtscts":
			opts = append(opts, serial.WithFlowControl(serial.FlowControlRTSCTS))
		}

		// Start the TUI
		if err := runListenTUI(portPath, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(listenCmd)

	// Add flags for serial configuration
	listenCmd.Flags().IntP("baud", "b", 115200, "Baud rate (default: 115200)")
	listenCmd.Flags().StringP("flow-control", "f", "none", "Flow control: none, cts, rtscts (default: none)")
}

// listenModel represents the Bubble Tea model for the listen command
type listenModel struct {
	port     *serial.Port
	portPath string
	viewport viewport.Model
	help     help.Model
	keys     listenKeyMap

	// State
	connected bool
	rawData   []dataReceivedMsg // Store raw data for reformatting
	data      []string          // Formatted display data
	status    string
	err       error
	ready     bool

	// Display modes
	showHex   bool
	showASCII bool

	// Cancellation and synchronization
	cancel context.CancelFunc
	ctx    context.Context
	mu     sync.RWMutex
}

type listenKeyMap struct {
	Quit      key.Binding
	Help      key.Binding
	Clear     key.Binding
	ToggleHex key.Binding
	ToggleASCII key.Binding
}

func (k listenKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Clear, k.ToggleHex, k.ToggleASCII, k.Quit}
}

func (k listenKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Help, k.Clear, k.ToggleHex, k.ToggleASCII, k.Quit},
	}
}

var defaultListenKeys = listenKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "Q", "ctrl+c"),
		key.WithHelp("q/ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Clear: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "clear buffer"),
	),
	ToggleHex: key.NewBinding(
		key.WithKeys("h"),
		key.WithHelp("h", "toggle hex"),
	),
	ToggleASCII: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "toggle ascii"),
	),
}

type dataReceivedMsg struct {
	timestamp time.Time
	data      []byte
}
type connectionStatusMsg struct {
	connected bool
	err       error
}

func runListenTUI(portPath string, opts ...serial.Option) error {
	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create initial model
	m := listenModel{
		portPath:  portPath,
		viewport:  viewport.New(80, 20),
		help:      help.New(),
		keys:      defaultListenKeys,
		rawData:   make([]dataReceivedMsg, 0),
		data:      make([]string, 0),
		status:    "Connecting...",
		showHex:   true,  // Show hex by default
		showASCII: true,  // Show ASCII by default
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start the TUI with alt screen and input handling
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Connect to serial port in background
	go func() {
		port, err := serial.Open(portPath, opts...)
		if err != nil {
			p.Send(connectionStatusMsg{connected: false, err: err})
			return
		}

		// Store port safely
		m.mu.Lock()
		m.port = port
		m.mu.Unlock()

		p.Send(connectionStatusMsg{connected: true, err: nil})

		// Start reading data with context cancellation
		go func() {
			defer func() {
				m.mu.Lock()
				if m.port != nil {
					m.port.Close()
					m.port = nil
				}
				m.mu.Unlock()
			}()

			buffer := make([]byte, 1024)
			for {
				n, err := port.ReadContext(ctx, buffer)
				if err != nil {
					// Context was cancelled or other error occurred
					return
				}
				if n > 0 {
					// Send raw data with timestamp - formatting will happen in Update method
					data := make([]byte, n)
					copy(data, buffer[:n])
					p.Send(dataReceivedMsg{
						timestamp: time.Now(),
						data:      data,
					})
				}
			}
		}()
	}()

	_, err := p.Run()

	// Ensure cleanup
	cancel()
	return err
}

func (m *listenModel) Init() tea.Cmd {
	return nil
}

// formatDataMessage formats received data according to display mode settings
func (m *listenModel) formatDataMessage(msg dataReceivedMsg) string {
	timestamp := msg.timestamp.Format("15:04:05.000")

	var parts []string

	if m.showHex {
		hexStr := fmt.Sprintf("% X", msg.data)
		parts = append(parts, fmt.Sprintf("HEX: %s", hexStr))
	}

	if m.showASCII {
		asciiStr := ""
		for _, b := range msg.data {
			if b >= 32 && b <= 126 {
				asciiStr += string(b)
			} else {
				asciiStr += "."
			}
		}
		parts = append(parts, fmt.Sprintf("ASCII: %s", asciiStr))
	}

	// If both are disabled, show raw bytes count
	if !m.showHex && !m.showASCII {
		parts = append(parts, fmt.Sprintf("BYTES: %d", len(msg.data)))
	}

	return fmt.Sprintf("[%s] %s", timestamp, strings.Join(parts, "  "))
}

// refreshDisplayData reformats all stored data according to current display settings
func (m *listenModel) refreshDisplayData() {
	m.data = make([]string, 0, len(m.rawData))
	for _, rawMsg := range m.rawData {
		m.data = append(m.data, m.formatDataMessage(rawMsg))
	}
	m.viewport.SetContent(strings.Join(m.data, "\n"))
	m.viewport.GotoBottom()
}

func (m *listenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		headerHeight := 3
		// Dynamic footer height based on help state
		footerHeight := 2
		if m.help.ShowAll {
			footerHeight = 4
		}
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

	case connectionStatusMsg:
		m.connected = msg.connected
		if msg.err != nil {
			m.err = msg.err
			m.status = fmt.Sprintf("Connection failed: %v", msg.err)
		} else {
			m.status = "Connected - listening for data..."
		}

	case dataReceivedMsg:
		m.rawData = append(m.rawData, msg)
		formattedMsg := m.formatDataMessage(msg)
		m.data = append(m.data, formattedMsg)
		m.viewport.SetContent(strings.Join(m.data, "\n"))
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
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

			return m, tea.Quit

		case key.Matches(msg, m.keys.Clear):
			m.rawData = make([]dataReceivedMsg, 0)
			m.data = make([]string, 0)
			m.viewport.SetContent("")

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, m.keys.ToggleHex):
			m.showHex = !m.showHex
			m.refreshDisplayData()

		case key.Matches(msg, m.keys.ToggleASCII):
			m.showASCII = !m.showASCII
			m.refreshDisplayData()
		}
	}

	// Update viewport only for messages it understands
	var cmd tea.Cmd
	switch msg.(type) {
	case tea.KeyMsg:
		// Don't pass key messages to viewport to prevent it from consuming them
	case dataReceivedMsg, connectionStatusMsg:
		// Don't pass our custom message types to viewport
	case tea.WindowSizeMsg:
		// Pass window resize messages to viewport
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *listenModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	statusStyle := lipgloss.NewStyle().
		Foreground(func() lipgloss.Color {
			if m.connected {
				return lipgloss.Color("40") // Green
			} else {
				return lipgloss.Color("196") // Red
			}
		}()).
		Bold(true)

	// Header
	title := titleStyle.Render(fmt.Sprintf("Serial Listen - %s", m.portPath))
	status := statusStyle.Render(m.status)
	header := lipgloss.JoinHorizontal(lipgloss.Left, title, " ", status)

	// Main content
	content := m.viewport.View()

	// Footer with help
	helpView := m.help.View(m.keys)

	// Layout with proper spacing
	contentWithBorder := lipgloss.NewStyle().BorderTop(true).BorderStyle(lipgloss.NormalBorder()).Render(content)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		contentWithBorder,
		helpView,
	)
}
