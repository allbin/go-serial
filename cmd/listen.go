/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/allbin/go-serial"
	"github.com/allbin/go-serial/internal/tui/components"
	"github.com/allbin/go-serial/internal/tui/keys"
	"github.com/allbin/go-serial/internal/tui/models"
	"github.com/allbin/go-serial/internal/tui/styles"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
  serial listen /dev/ttyUSB0 --flow-control cts --initial-rts`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		portPath := args[0]

		// Get flags
		baudRate, _ := cmd.Flags().GetInt("baud")
		flowControl, _ := cmd.Flags().GetString("flow-control")
		initialRTS, _ := cmd.Flags().GetBool("initial-rts")
		noTimestamps, _ := cmd.Flags().GetBool("no-timestamps")
		showIndicators, _ := cmd.Flags().GetBool("show-indicators")
		rawMode, _ := cmd.Flags().GetBool("raw")

		// Configure port options
		opts := []serial.Option{
			serial.WithBaudRate(baudRate),
		}

		switch strings.ToLower(flowControl) {
		case "cts":
			opts = append(opts, serial.WithFlowControl(serial.FlowControlCTS))
			if initialRTS {
				opts = append(opts, serial.WithInitialRTS(true))
			}
		case "rtscts":
			opts = append(opts, serial.WithFlowControl(serial.FlowControlRTSCTS))
			if initialRTS {
				opts = append(opts, serial.WithInitialRTS(true))
			}
		}

		// Start the TUI
		if err := runListenTUI(portPath, noTimestamps, showIndicators, rawMode, opts...); err != nil {
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
	listenCmd.Flags().Bool("initial-rts", false, "Assert RTS on port open (required for CTS flow control)")

	// Add flags for display formatting
	listenCmd.Flags().Bool("no-timestamps", false, "Hide timestamps from output")
	listenCmd.Flags().Bool("show-indicators", false, "Show RX/TX indicators (off by default)")
	listenCmd.Flags().Bool("raw", false, "Raw output mode: no timestamps, no indicators")
}

// listenModel represents the Bubble Tea model for the listen command
type listenModel struct {
	*models.SerialModel
	terminal  *components.Terminal
	statusBar *components.StatusBar
	help      help.Model
	keys      keys.TerminalKeys
}

func runListenTUI(portPath string, noTimestamps, showIndicators, rawMode bool, opts ...serial.Option) error {

	// Create configuration from options to show in status bar
	config := serial.DefaultConfig()
	for _, opt := range opts {
		opt(&config)
	}

	// Create connection info for status bar
	connInfo := &components.ConnectionInfo{
		BaudRate:    config.BaudRate,
		FlowControl: config.FlowControl,
		DataBits:    config.DataBits,
		StopBits:    config.StopBits,
		Parity:      config.Parity,
	}

	// Create initial model
	serialModel := models.NewSerialModel(portPath)
	terminal := components.NewTerminal(80, 20)

	// Configure formatting options
	// Default: no indicators, show timestamps
	if rawMode {
		terminal.SetFormatOptions(true, true) // No timestamps, no indicators
	} else if noTimestamps {
		terminal.SetFormatOptions(true, !showIndicators) // No timestamps, indicators based on flag
	} else {
		terminal.SetFormatOptions(false, !showIndicators) // Show timestamps, indicators based on flag
	}

	m := listenModel{
		SerialModel: serialModel,
		terminal:    terminal,
		statusBar:   components.NewStatusBar("Serial Listen", portPath),
		help:        help.New(),
		keys:        keys.NewTerminalKeys(),
	}
	m.statusBar.SetConnecting()
	m.statusBar.SetConnectionInfo(connInfo)

	// Start the TUI with alt screen and input handling
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Connect to serial port in background
	go func() {
		port, err := serial.Open(portPath, opts...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to open port: %v\n", err)
			p.Send(models.ConnectionStatusMsg{Connected: false, Error: err})
			return
		}

		// Store port safely
		m.SetPort(port)

		p.Send(models.ConnectionStatusMsg{Connected: true, Error: nil})

		// Start reading data with context cancellation
		go func() {
			defer func() {
				// Only close the port when this goroutine exits, don't cleanup the whole model
				if port != nil {
					port.Close()
				}
			}()

			buffer := make([]byte, 4096)
			for {
				select {
				case <-m.GetContext().Done():
					// Context was cancelled, exit cleanly
					return
				default:
					// Try to read data from the serial port
					n, err := port.ReadContext(m.GetContext(), buffer)
					if err != nil {
						// Check if it's a context cancellation
						if m.GetContext().Err() != nil {
							return // Context cancelled, exit cleanly
						}
						// For other errors, continue trying to read
						continue
					}
					if n > 0 {
						// Debug: Log raw bytes received to file
						if debugFile, err := os.OpenFile("serial_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
							fmt.Fprintf(debugFile, "[SERIAL_RX] Read %d bytes: %X\n", n, buffer[:n])
							debugFile.Close()
						}

						// Send raw data with timestamp - formatting will happen in Update method
						data := make([]byte, n)
						copy(data, buffer[:n])
						p.Send(components.DataReceivedMsg{
							Timestamp: time.Now(),
							Data:      data,
						})
					}
				}
			}
		}()
	}()

	_, err := p.Run()

	// Ensure cleanup
	m.Cancel()
	return err
}

func (m *listenModel) Init() tea.Cmd {
	return nil
}

func (m *listenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Status bar is single line
		statusBarHeight := 1
		verticalMarginHeight := statusBarHeight

		if !m.IsReady() {
			m.terminal.SetSize(msg.Width, msg.Height-verticalMarginHeight)
			m.SetReady(true)
		} else {
			m.terminal.SetSize(msg.Width, msg.Height-verticalMarginHeight)
		}
		m.statusBar.SetWidth(msg.Width)

	case models.ConnectionStatusMsg:
		m.SetConnected(msg.Connected)
		if msg.Error != nil {
			m.SetError(msg.Error)
			m.statusBar.SetDisconnected(msg.Error)
		} else {
			m.statusBar.SetConnected()
		}

	case components.DataReceivedMsg:
		// Safely handle the data message
		defer func() {
			if r := recover(); r != nil {
				// If there's a panic in data handling, don't crash the whole UI
				// Just continue running
			}
		}()

		// Ensure we're ready to display data - if window size hasn't been set yet,
		// use reasonable defaults
		if !m.IsReady() {
			m.terminal.SetSize(80, 20) // Default terminal size
			m.SetReady(true)
		}

		m.AddRawData(msg)
		m.terminal.AddMessage(msg)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			m.Cleanup()
			return m, tea.Quit

		case key.Matches(msg, m.keys.Clear):
			m.ClearData()
			m.terminal.Clear()

		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll

		case key.Matches(msg, m.keys.ToggleHex):
			m.terminal.ToggleHex()
			m.terminal.RefreshDisplayWithRawData(m.GetRawData())

		case key.Matches(msg, m.keys.ToggleASCII):
			m.terminal.ToggleASCII()
			m.terminal.RefreshDisplayWithRawData(m.GetRawData())

		case key.Matches(msg, m.keys.ToggleTimestamps):
			m.terminal.ToggleTimestamps()
			m.terminal.RefreshDisplayWithRawData(m.GetRawData())

		case key.Matches(msg, m.keys.ToggleIndicators):
			m.terminal.ToggleIndicators()
			m.terminal.RefreshDisplayWithRawData(m.GetRawData())
		}
	}

	// Update terminal viewport for window resize messages
	var cmd tea.Cmd
	switch msg.(type) {
	case tea.WindowSizeMsg:
		_, cmd = m.terminal.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *listenModel) View() string {
	// Always show the UI, even if not fully ready
	// If not ready, we'll show what we can with defaults

	// Main content (no header now)
	var content string
	if m.IsReady() {
		content = m.terminal.View()
	} else {
		// Show initializing message in a consistent format
		content = "Initializing..."
	}

	// Comprehensive status bar (listen mode is always NORMAL, no sending mode)
	inputMode := "NORMAL"
	sendingMode := "LISTEN" // Special mode for listen-only
	timestamp := time.Now().Format("15:04:05")

	// Set the status bar width to match terminal
	terminalWidth := 80
	if m.IsReady() {
		terminalWidth = m.terminal.GetViewport().Width
	}
	m.statusBar.SetWidth(terminalWidth)

	statusBar := m.statusBar.ComprehensiveStatusBar(inputMode, sendingMode, "FOLLOW", m.IsConnected(), timestamp)

	// Layout without header, with comprehensive status bar at bottom
	contentWithBorder := styles.ContentBorderStyle.Render(content)

	// Show help if requested
	var helpView string
	if m.help.ShowAll {
		helpView = m.help.View(m.keys)
		helpStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			Margin(1, 0)
		helpView = helpStyle.Render(helpView)
	}

	if m.help.ShowAll {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			contentWithBorder,
			helpView,
			statusBar,
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		contentWithBorder,
		statusBar,
	)
}
