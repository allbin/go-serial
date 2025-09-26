/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/allbin/serial"
	"github.com/allbin/serial/internal/tui/components"
	"github.com/allbin/serial/internal/tui/keys"
	"github.com/allbin/serial/internal/tui/models"
	"github.com/allbin/serial/internal/tui/styles"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect <port>",
	Short: "Connect to a serial port with bidirectional communication",
	Long: `Connect to a serial port with a beautiful bidirectional terminal interface.

This command opens the specified serial port and provides an interactive terminal
with real-time bidirectional communication. Features include:
- Real-time data streaming with timestamps
- Input field for sending data
- ASCII and hex display modes
- Connection status indicators
- Configurable baud rate and flow control
- CTS flow control monitoring and debugging
- Configurable CTS timeout handling
- Clean, responsive interface

Example usage:
  serial connect /dev/ttyUSB0
  serial connect /dev/ttyUSB0 --baud 9600
  serial connect /dev/ttyUSB0 --flow-control cts --cts-timeout 1000`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		portPath := args[0]

		// Get flags
		baudRate, _ := cmd.Flags().GetInt("baud")
		flowControl, _ := cmd.Flags().GetString("flow-control")
		ctsTimeoutMs, _ := cmd.Flags().GetInt("cts-timeout")
		syncWrites, _ := cmd.Flags().GetBool("sync-writes")

		// Configure port options
		opts := []serial.Option{
			serial.WithBaudRate(baudRate),
			serial.WithCTSTimeout(time.Duration(ctsTimeoutMs) * time.Millisecond),
		}

		// Configure write mode
		if syncWrites {
			fmt.Fprintf(os.Stderr, "[DEBUG] Sync writes enabled via flag\n")
			opts = append(opts, serial.WithSyncWrite())
		} else {
			fmt.Fprintf(os.Stderr, "[DEBUG] Sync writes disabled (default buffered)\n")
		}

		switch strings.ToLower(flowControl) {
		case "cts":
			opts = append(opts, serial.WithFlowControl(serial.FlowControlCTS))
		case "rtscts":
			opts = append(opts, serial.WithFlowControl(serial.FlowControlRTSCTS))
		}

		// Start the TUI
		if err := runConnectTUI(portPath, opts...); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)

	// Add flags for serial configuration
	connectCmd.Flags().IntP("baud", "b", 115200, "Baud rate (default: 115200)")
	connectCmd.Flags().StringP("flow-control", "f", "none", "Flow control: none, cts, rtscts (default: none)")
	connectCmd.Flags().IntP("cts-timeout", "t", 500, "CTS timeout in milliseconds (default: 500)")
	connectCmd.Flags().Bool("sync-writes", false, "Enable synchronous writes (O_SYNC) for guaranteed transmission")
}

// connectModel represents the Bubble Tea model for the connect command
type connectModel struct {
	*models.SerialModel
	terminal  *components.Terminal
	statusBar *components.StatusBar
	input     *components.Input
	help      help.Model
	keys      keys.ConnectKeys
}

func runConnectTUI(portPath string, opts ...serial.Option) error {
	fmt.Fprintf(os.Stderr, "[DEBUG] Starting connect TUI\n")

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
		CTSEnabled:  config.FlowControl == serial.FlowControlCTS || config.FlowControl == serial.FlowControlRTSCTS,
		CTSStatus:   false, // Will be updated when port opens
	}

	// Create initial model with minimal dimensions - let WindowSizeMsg set proper size
	serialModel := models.NewSerialModel(portPath)
	m := connectModel{
		SerialModel: serialModel,
		terminal:    components.NewTerminal(0, 0), // Will be properly sized by WindowSizeMsg
		statusBar:   components.NewStatusBar("Serial Connect", portPath),
		input:       components.NewInput("Type message and press Enter to send..."),
		help:        help.New(),
		keys:        keys.NewConnectKeys(),
	}
	m.statusBar.SetConnecting()
	m.statusBar.SetConnectionInfo(connInfo)

	// Start the TUI with alt screen and input handling
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Connect to serial port in background
	go func() {
		port, err := serial.Open(portPath, opts...)
		if err != nil {
			p.Send(models.ConnectionStatusMsg{Connected: false, Error: err})
			return
		}

		// Store port safely
		m.SetPort(port)

		p.Send(models.ConnectionStatusMsg{Connected: true, Error: nil})

		// Start CTS monitoring if CTS flow control is enabled
		if connInfo.CTSEnabled {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						// Silently handle any panics in CTS monitoring
					}
				}()

				var lastCTSStatus bool = false
				ticker := time.NewTicker(10 * time.Microsecond) // Check CTS every 10μs for Neocortec timing
				defer ticker.Stop()

				for {
					select {
					case <-m.GetContext().Done():
						return
					case <-ticker.C:
						ctsStatus, err := port.GetCTSStatus()
						if err != nil {
							// If we can't read CTS, stop monitoring
							return
						}
						if ctsStatus != lastCTSStatus {
							// CTS status changed, notify
							p.Send(components.CTSStatusMsg{
								Status:    ctsStatus,
								Timestamp: time.Now(),
							})
							lastCTSStatus = ctsStatus
						}
					}
				}
			}()
		}

		// Start reading data with context cancellation
		go func() {
			defer func() {
				// Only close the port when this goroutine exits
				if port != nil {
					port.Close()
				}
			}()

			buffer := make([]byte, 1024)
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

func (m *connectModel) Init() tea.Cmd {
	return nil
}

// parseHexInput converts hex strings to bytes. Supports both:
// - Space-separated: "48 65 6C 6C 6F"
// - Continuous: "48656C6C6F"
func parseHexInput(hexStr string) ([]byte, error) {
	// Remove any spaces and convert to uppercase for consistency
	cleanHex := strings.ReplaceAll(strings.TrimSpace(hexStr), " ", "")
	if len(cleanHex) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	// Check if it's valid hex characters
	for _, char := range cleanHex {
		if !((char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f')) {
			return nil, fmt.Errorf("invalid hex character '%c'", char)
		}
	}

	// Must be even number of hex digits to form complete bytes
	if len(cleanHex)%2 != 0 {
		return nil, fmt.Errorf("hex string must have even number of digits (got %d)", len(cleanHex))
	}

	// Parse pairs of hex digits into bytes
	bytes := make([]byte, 0, len(cleanHex)/2)
	for i := 0; i < len(cleanHex); i += 2 {
		hexByte := cleanHex[i : i+2]
		b, err := strconv.ParseUint(hexByte, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid hex byte '%s': %v", hexByte, err)
		}
		bytes = append(bytes, byte(b))
	}
	return bytes, nil
}

func (m *connectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Input area height (includes border)
		inputHeight := 3
		// Status bar is single line
		statusBarHeight := 1
		verticalMarginHeight := inputHeight + statusBarHeight

		if !m.IsReady() {
			m.terminal.SetSize(msg.Width, msg.Height-verticalMarginHeight)
			m.input.SetWidth(msg.Width)
			m.SetReady(true)
		} else {
			m.terminal.SetSize(msg.Width, msg.Height-verticalMarginHeight)
			m.input.SetWidth(msg.Width)
		}
		m.statusBar.SetWidth(msg.Width)

	case models.ConnectionStatusMsg:
		m.SetConnected(msg.Connected)
		if msg.Error != nil {
			m.SetError(msg.Error)
			m.statusBar.SetDisconnected(msg.Error)
		} else {
			m.statusBar.SetConnected()
			m.input.Focus()
		}

	case components.CTSStatusMsg:
		// Update CTS status in status bar
		m.statusBar.UpdateCTSStatus(msg.Status)

		// Add CTS status change to terminal with timestamp
		if m.IsReady() {
			statusText := "CTS: OFF"
			if msg.Status {
				statusText = "CTS: ON"
			}
			ctsData := components.DataReceivedMsg{
				Timestamp: msg.Timestamp,
				Data:      []byte(statusText),
				IsTX:      false,
			}
			m.terminal.AddMessage(ctsData)
		}

	case components.DataReceivedMsg:
		// Safely handle the data message
		defer func() {
			if r := recover(); r != nil {
				// If there's a panic in data handling, don't crash the whole UI
				// Just continue running
			}
		}()

		// Only process data if we're ready (WindowSizeMsg has been received)
		if m.IsReady() {
			m.AddRawData(msg)
			m.terminal.AddMessage(msg)
		}

	case tea.KeyMsg:
		// Handle mode-specific keys
		if m.IsInInsertMode() {
			// Insert mode - handle input and escape
			switch {
			case key.Matches(msg, m.keys.Escape):
				m.SetInputMode(models.InputModeNormal)
				m.input.Blur()
				return m, tea.Batch(cmds...)
			case key.Matches(msg, m.keys.Enter):
				// Send the message
				port := m.GetPort()
				if m.input.Value() != "" && port != nil {
					inputStr := m.input.Value()
					var dataToSend []byte
					var displayData []byte
					var err error

					switch m.input.GetSendingMode() {
					case components.SendingModeASCII:
						dataToSend = []byte(inputStr + "\n")
						displayData = []byte(inputStr)
					case components.SendingModeHex:
						dataToSend, err = parseHexInput(inputStr)
						if err != nil {
							// Show error in terminal but don't send anything
							errorMsg := fmt.Sprintf("Invalid hex input: %v", err)
							timestamp := time.Now()
							errorData := components.DataReceivedMsg{
								Timestamp: timestamp,
								Data:      []byte(errorMsg),
								IsTX:      false,
							}
							m.terminal.AddMessage(errorData)
							return m, tea.Batch(cmds...)
						}
						displayData = dataToSend
					}

					// Send the data with proper timeout handling and status updates
					writeStatusCh := make(chan error, 1)
					transmittingStatusCh := make(chan bool, 1)

					go func(port *serial.Port, dataToSend []byte) {
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()

						// Signal that we're about to start writing (may block on CTS)
						transmittingStatusCh <- true

						_, err := port.WriteContext(ctx, dataToSend)
						writeStatusCh <- err
						close(writeStatusCh)
					}(port, dataToSend)

					// Return commands for both status updates
					cmds = append(cmds, func() tea.Msg {
						// Wait for write to start - show TRANSMITTING status
						<-transmittingStatusCh
						return components.DataReceivedMsg{
							Timestamp: time.Now(),
							Data:      displayData,
							IsTX:      true,
							Status:    "TRANSMITTING",
						}
					})

					cmds = append(cmds, func() tea.Msg {
						err := <-writeStatusCh
						// Send completion status
						finalStatus := components.DataReceivedMsg{
							Timestamp: time.Now(),
							Data:      displayData,
							IsTX:      true,
						}
						if err != nil {
							finalStatus.Status = "ERROR"
						} else {
							finalStatus.Status = "WRITTEN"
						}
						return finalStatus
					})

					// Add to display with TX prefix (initially as PENDING)
					timestamp := time.Now()
					txData := components.DataReceivedMsg{
						Timestamp: timestamp,
						Data:      displayData,
						IsTX:      true,
						Status:    "PENDING",
					}
					// Add to both raw data store and terminal display
					m.AddRawData(txData)
					m.terminal.AddMessage(txData)

					// Add to history before clearing
					m.input.AddToHistory(inputStr)
					m.input.SetValue("")
				}
				return m, tea.Batch(cmds...)
			case key.Matches(msg, m.keys.Up):
				m.input.NavigateHistoryUp()
				return m, tea.Batch(cmds...)
			case key.Matches(msg, m.keys.Down):
				m.input.NavigateHistoryDown()
				return m, tea.Batch(cmds...)
			case key.Matches(msg, m.keys.ToggleSendMode):
				m.input.ToggleSendingMode()
				return m, tea.Batch(cmds...)
			}
		} else {
			// Normal mode - handle navigation and mode switching
			switch {
			case key.Matches(msg, m.keys.Quit):
				m.Cleanup()
				return m, tea.Quit

			case key.Matches(msg, m.keys.InsertMode):
				m.SetInputMode(models.InputModeInsert)
				m.input.Focus()
				return m, tea.Batch(cmds...)

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

			case key.Matches(msg, m.keys.ToggleSendMode):
				m.input.ToggleSendingMode()
			}
		}
	}

	// Update components (only update input in insert mode)
	var cmd tea.Cmd
	if m.IsInInsertMode() {
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update terminal viewport for window resize messages
	switch msg.(type) {
	case tea.WindowSizeMsg:
		_, cmd = m.terminal.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *connectModel) View() string {
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

	// Input area
	inputMode := m.GetInputMode().String()
	isInsertMode := m.IsInInsertMode()
	input := m.input.ViewWithMode(inputMode, isInsertMode)

	// Comprehensive status bar with all info
	sendingMode := m.input.GetSendingMode().String()
	timestamp := time.Now().Format("15:04:05")

	// Set the status bar width to match terminal
	terminalWidth := 80
	if m.IsReady() {
		terminalWidth = m.terminal.GetViewport().Width
	}
	m.statusBar.SetWidth(terminalWidth)

	statusBar := m.statusBar.ComprehensiveStatusBar(inputMode, sendingMode, m.IsConnected(), timestamp)

	// Layout without header, with comprehensive status bar at bottom
	contentWithBorder := styles.ContentBorderStyle.Render(content)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		contentWithBorder,
		input,
		statusBar,
	)
}
