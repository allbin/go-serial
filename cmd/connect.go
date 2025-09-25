/*
Copyright © 2025 Mathias Djärv <mathias.djarv@allbinary.se>
*/
package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mdjarv/serial"
	"github.com/spf13/cobra"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect [port]",
	Short: "Interactive serial terminal with port selection",
	Long: `Open an interactive serial terminal with beautiful port selection interface.

This command provides a full-featured serial terminal experience with:
- Interactive port selection (if no port specified)
- Real-time bidirectional communication
- Split-pane interface (receive/send)
- Connection status indicators
- Configurable baud rate and flow control
- Clean, responsive terminal UI

If no port is specified, an interactive port selector will be shown.
Otherwise, connects directly to the specified port.

Example usage:
  serial connect                    # Interactive port selection
  serial connect /dev/ttyUSB0       # Connect directly to port
  serial connect /dev/ttyUSB0 --baud 9600 --flow-control cts`,
	Run: func(cmd *cobra.Command, args []string) {
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

		var portPath string
		if len(args) > 0 {
			portPath = args[0]
		}

		// Start the interactive TUI
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
}

// connectState represents the different states of the connect interface
type connectState int

const (
	statePortSelection connectState = iota
	stateConnecting
	stateTerminal
	stateError
)

// connectModel represents the Bubble Tea model for the connect command
type connectModel struct {
	state     connectState
	portList  list.Model
	textInput textinput.Model
	viewport  viewport.Model
	help      help.Model
	keys      connectKeyMap

	// Serial communication
	port        *serial.Port
	portPath    string
	portOptions []serial.Option

	// State
	ready     bool
	connected bool
	data      []string
	status    string
	err       error

	// Terminal dimensions
	width  int
	height int
}

type connectKeyMap struct {
	Quit  key.Binding
	Help  key.Binding
	Enter key.Binding
	Up    key.Binding
	Down  key.Binding
	Send  key.Binding
}

func (k connectKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Send, k.Quit}
}

func (k connectKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Help, k.Send, k.Quit},
	}
}

var defaultConnectKeys = connectKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select/send"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Send: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "send message"),
	),
}

// portItem represents a port in the selection list
type portItem struct {
	path        string
	description string
	portType    string
}

func (i portItem) FilterValue() string { return i.path }
func (i portItem) Title() string       { return i.path }
func (i portItem) Description() string { return fmt.Sprintf("%s - %s", i.portType, i.description) }

type connectPortSelectedMsg string
type connectDataReceivedMsg string
type connectConnectionStatusMsg struct {
	connected bool
	err       error
}
type connectTerminalDataMsg string

func runConnectTUI(initialPort string, opts ...serial.Option) error {
	// Create the model
	m := newConnectModel(initialPort, opts...)

	// Start the TUI
	p := tea.NewProgram(&m, tea.WithAltScreen())

	_, err := p.Run()
	return err
}

func newConnectModel(initialPort string, opts ...serial.Option) connectModel {
	// Create text input for terminal
	ti := textinput.New()
	ti.Placeholder = "Type message and press Enter to send..."
	ti.CharLimit = 256

	// Create viewport for received data
	vp := viewport.New(80, 10)

	// Create help
	h := help.New()
	h.ShowAll = false

	m := connectModel{
		textInput:   ti,
		viewport:    vp,
		help:        h,
		keys:        defaultConnectKeys,
		portOptions: opts,
		data:        make([]string, 0),
		status:      "Initializing...",
	}

	if initialPort != "" {
		// Connect directly to specified port
		m.portPath = initialPort
		m.state = stateConnecting
	} else {
		// Show port selection
		m.state = statePortSelection
		m.setupPortList()
	}

	return m
}

func (m *connectModel) setupPortList() {
	// Get available ports
	ports, err := serial.ListPorts()
	if err != nil {
		m.state = stateError
		m.err = err
		return
	}

	if len(ports) == 0 {
		m.state = stateError
		m.err = fmt.Errorf("no serial ports found")
		return
	}

	// Create port items
	items := make([]list.Item, len(ports))
	for i, port := range ports {
		info, err := serial.GetPortInfo(port)
		if err != nil {
			items[i] = portItem{
				path:        port,
				description: "Unknown",
				portType:    "Serial Port",
			}
			continue
		}

		items[i] = portItem{
			path:        info.Name,
			description: info.Description,
			portType:    getPortType(info.Name),
		}
	}

	// Setup list
	l := list.New(items, list.NewDefaultDelegate(), 80, 20)
	l.Title = "Select a Serial Port"
	l.SetFilteringEnabled(true)

	m.portList = l
}

func (m *connectModel) Init() tea.Cmd {
	switch m.state {
	case stateConnecting:
		return m.connectToPort()
	case statePortSelection:
		return nil
	default:
		return nil
	}
}

func (m *connectModel) connectToPort() tea.Cmd {
	return func() tea.Msg {
		port, err := serial.Open(m.portPath, m.portOptions...)
		if err != nil {
			return connectConnectionStatusMsg{connected: false, err: err}
		}

		// Store the port connection
		m.port = port

		return connectConnectionStatusMsg{connected: true, err: nil}
	}
}

func (m *connectModel) startSerialReader() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		if m.port == nil {
			return nil
		}

		// Try to read data from the serial port (non-blocking)
		buffer := make([]byte, 256)
		n, err := m.port.Read(buffer)
		if err != nil || n == 0 {
			return nil
		}

		// Convert to string and send as message
		data := strings.TrimRight(string(buffer[:n]), "\r\n")
		if data != "" {
			return connectDataReceivedMsg(data)
		}

		return nil
	})
}

func (m *connectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		switch m.state {
		case statePortSelection:
			m.portList.SetWidth(msg.Width)
			m.portList.SetHeight(msg.Height - 4)
		case stateTerminal:
			if !m.ready {
				// Terminal mode: split screen (70% viewport, 30% input area)
				viewportHeight := int(float64(msg.Height)*0.7) - 4
				m.viewport = viewport.New(msg.Width, viewportHeight)
				m.textInput.Width = msg.Width - 4
				m.ready = true
			} else {
				viewportHeight := int(float64(m.height)*0.7) - 4
				m.viewport.Width = msg.Width
				m.viewport.Height = viewportHeight
				m.textInput.Width = msg.Width - 4
			}
		}

	case connectConnectionStatusMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
		} else {
			m.state = stateTerminal
			m.connected = true
			m.status = fmt.Sprintf("Connected to %s", m.portPath)
			m.textInput.Focus()

			// Start background reader
			return m, m.startSerialReader()
		}

	case connectDataReceivedMsg:
		timestamp := time.Now().Format("15:04:05")
		m.data = append(m.data, fmt.Sprintf("[%s] RX: %s", timestamp, string(msg)))
		m.viewport.SetContent(strings.Join(m.data, "\n"))
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		switch m.state {
		case statePortSelection:
			switch {
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.Enter):
				if selectedItem := m.portList.SelectedItem(); selectedItem != nil {
					if port, ok := selectedItem.(portItem); ok {
						m.portPath = port.path
						m.state = stateConnecting
						m.status = fmt.Sprintf("Connecting to %s...", port.path)
						return m, m.connectToPort()
					}
				}
			}

		case stateTerminal:
			switch {
			case key.Matches(msg, m.keys.Quit):
				if m.port != nil {
					m.port.Close()
				}
				return m, tea.Quit
			case key.Matches(msg, m.keys.Enter):
				// Send the message
				if m.textInput.Value() != "" && m.port != nil {
					message := m.textInput.Value()
					go func() {
						m.port.Write([]byte(message + "\n"))
					}()

					// Add to display
					timestamp := time.Now().Format("15:04:05")
					m.data = append(m.data, fmt.Sprintf("[%s] TX: %s", timestamp, message))
					m.viewport.SetContent(strings.Join(m.data, "\n"))
					m.viewport.GotoBottom()

					m.textInput.SetValue("")
				}
			case key.Matches(msg, m.keys.Help):
				m.help.ShowAll = !m.help.ShowAll
			}

		case stateError:
			if key.Matches(msg, m.keys.Quit) {
				return m, tea.Quit
			}
		}
	}

	// Update components based on state
	var cmd tea.Cmd
	switch m.state {
	case statePortSelection:
		m.portList, cmd = m.portList.Update(msg)
		cmds = append(cmds, cmd)

	case stateTerminal:
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)

		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *connectModel) View() string {
	switch m.state {
	case statePortSelection:
		return m.renderPortSelection()
	case stateConnecting:
		return m.renderConnecting()
	case stateTerminal:
		return m.renderTerminal()
	case stateError:
		return m.renderError()
	default:
		return "Unknown state"
	}
}

func (m *connectModel) renderPortSelection() string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.portList.View(),
		m.help.View(m.keys),
	)
}

func (m *connectModel) renderConnecting() string {
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Align(lipgloss.Center).
		Width(m.width)

	return style.Render(fmt.Sprintf("🔌 %s", m.status))
}

func (m *connectModel) renderTerminal() string {
	if !m.ready {
		return "\n  Initializing terminal..."
	}

	// Styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		Background(lipgloss.Color("235")).
		Padding(0, 1).
		Width(m.width)

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	// Header
	header := headerStyle.Render(fmt.Sprintf("📡 Serial Terminal - %s", m.portPath))

	// Main content (received data)
	content := m.viewport.View()

	// Input area
	input := inputStyle.Render(m.textInput.View())

	// Help
	helpView := m.help.View(m.keys)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		lipgloss.NewStyle().BorderTop(true).BorderStyle(lipgloss.NormalBorder()).Render(content),
		input,
		helpView,
	)
}

func (m *connectModel) renderError() string {
	errorStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("196")).
		Align(lipgloss.Center).
		Width(m.width)

	return errorStyle.Render(fmt.Sprintf("❌ Error: %v\n\nPress any key to exit", m.err))
}
