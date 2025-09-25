package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/mdjarv/serial"
	"github.com/mdjarv/serial/internal/tui/colors"
	"github.com/mdjarv/serial/internal/tui/styles"
)

// CTSStatusMsg represents a CTS status change
type CTSStatusMsg struct {
	Status    bool
	Timestamp time.Time
}

type ConnectionInfo struct {
	BaudRate    int
	FlowControl serial.FlowControl
	DataBits    int
	StopBits    int
	Parity      serial.Parity
	CTSStatus   bool
	CTSEnabled  bool
}

type StatusBar struct {
	title          string
	portPath       string
	status         string
	err            error
	width          int
	connectionInfo *ConnectionInfo
}

func NewStatusBar(title, portPath string) *StatusBar {
	return &StatusBar{
		title:    title,
		portPath: portPath,
		status:   "Initializing...",
	}
}

func (sb *StatusBar) SetStatus(status string, err error) {
	sb.status = status
	sb.err = err
}

func (sb *StatusBar) SetWidth(width int) {
	sb.width = width
}

func (sb *StatusBar) SetConnectionInfo(info *ConnectionInfo) {
	sb.connectionInfo = info
}

func (sb *StatusBar) UpdateCTSStatus(ctsStatus bool) {
	if sb.connectionInfo != nil {
		sb.connectionInfo.CTSStatus = ctsStatus
	}
}

func (sb *StatusBar) SetConnecting() {
	sb.status = "Connecting..."
	sb.err = nil
}

func (sb *StatusBar) SetConnected() {
	sb.status = "Connected - listening for data..."
	sb.err = nil
}

func (sb *StatusBar) SetDisconnected(err error) {
	if err != nil {
		sb.status = fmt.Sprintf("Connection failed: %v", err)
		sb.err = err
	} else {
		sb.status = "Disconnected"
		sb.err = nil
	}
}

func flowControlToString(fc serial.FlowControl) string {
	switch fc {
	case serial.FlowControlNone:
		return "None"
	case serial.FlowControlCTS:
		return "CTS"
	case serial.FlowControlRTSCTS:
		return "RTS/CTS"
	default:
		return "Unknown"
	}
}

func parityToString(p serial.Parity) string {
	switch p {
	case serial.ParityNone:
		return "N"
	case serial.ParityEven:
		return "E"
	case serial.ParityOdd:
		return "O"
	case serial.ParityMark:
		return "M"
	case serial.ParitySpace:
		return "S"
	default:
		return "N"
	}
}

func (sb *StatusBar) ViewAsHeader(connected bool) string {
	// This is the old header view, kept for compatibility if needed
	title := styles.TitleStyle.Render(sb.portPath)

	var connectionInfo string
	if sb.connectionInfo != nil {
		connectionInfo = fmt.Sprintf(" | %d baud, %d%s%d, flow: %s",
			sb.connectionInfo.BaudRate,
			sb.connectionInfo.DataBits,
			parityToString(sb.connectionInfo.Parity),
			sb.connectionInfo.StopBits,
			flowControlToString(sb.connectionInfo.FlowControl))
	}

	connInfoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Faint(true)
	connInfo := connInfoStyle.Render(connectionInfo)

	return lipgloss.JoinHorizontal(lipgloss.Left, title, connInfo)
}

// ComprehensiveStatusBar renders a comprehensive status bar with all connection info
func (sb *StatusBar) ComprehensiveStatusBar(inputMode, sendingMode string, connected bool, timestamp string) string {
	terminalWidth := sb.width
	if terminalWidth <= 0 {
		terminalWidth = 80
	}

	// Section 1: Mode indicator (like NORMAL in nvim)
	var modeStyle lipgloss.Style
	var modeText string
	if inputMode == "INSERT" {
		modeStyle = lipgloss.NewStyle().
			Foreground(colors.Base).
			Background(colors.Green).
			Bold(true).
			Padding(0, 1)
		modeText = "INSERT"
	} else {
		modeStyle = lipgloss.NewStyle().
			Foreground(colors.Base).
			Background(colors.Blue).
			Bold(true).
			Padding(0, 1)
		modeText = "NORMAL"
	}
	mode := modeStyle.Render(modeText)

	// Section 2: Port path with connection indicator
	portStyle := lipgloss.NewStyle().
		Foreground(colors.Mauve).
		Bold(true).
		Padding(0, 1)
	port := portStyle.Render(sb.portPath)

	// Section 3: Single character connection indicator
	var connIndicator string
	var connStyle lipgloss.Style

	if sb.err != nil {
		connStyle = lipgloss.NewStyle().Foreground(colors.Red)
		connIndicator = "✗"
	} else if connected {
		connStyle = lipgloss.NewStyle().Foreground(colors.Green)
		connIndicator = "●"
	} else if sb.status == "Connecting..." {
		connStyle = lipgloss.NewStyle().Foreground(colors.Yellow)
		connIndicator = "○"
	} else {
		connStyle = lipgloss.NewStyle().Foreground(colors.Red)
		connIndicator = "○"
	}

	connectionIndicator := connStyle.Render(connIndicator)

	// Section 4: Connection info (like file type with icon)
	var connInfo string
	if sb.connectionInfo != nil {
		ctsInfo := ""
		if sb.connectionInfo.CTSEnabled {
			if sb.connectionInfo.CTSStatus {
				ctsInfo = " CTS:✓"
			} else {
				ctsInfo = " CTS:✗"
			}
		}
		connInfo = fmt.Sprintf("⚡ %d baud %d%s%d %s%s",
			sb.connectionInfo.BaudRate,
			sb.connectionInfo.DataBits,
			parityToString(sb.connectionInfo.Parity),
			sb.connectionInfo.StopBits,
			flowControlToString(sb.connectionInfo.FlowControl),
			ctsInfo)
	} else {
		connInfo = "⚡ serial"
	}
	connInfoStyle := lipgloss.NewStyle().
		Foreground(colors.Subtext0).
		Padding(0, 1)
	connectionDetails := connInfoStyle.Render(connInfo)

	// Section 5: Timestamp (like position)
	timeStyle := lipgloss.NewStyle().
		Foreground(colors.Subtext1).
		Padding(0, 1)
	time := timeStyle.Render(timestamp)

	// Create muted divider
	dividerStyle := lipgloss.NewStyle().
		Foreground(colors.Surface2).
		Padding(0, 1)
	divider := dividerStyle.Render("│")

	// Section: Sending mode indicator with Tab hint (only show in INSERT mode)
	var sendingModeInfo string
	if inputMode == "INSERT" {
		sendingModeStyle := lipgloss.NewStyle().
			Foreground(colors.Peach).
			Bold(true).
			Padding(0, 1)
		sendingModeInfo = sendingModeStyle.Render(fmt.Sprintf("[%s] Tab to toggle", sendingMode))
	}

	// Build left side: mode (no divider) port + connection indicator, sending mode, then divider
	var leftSide string
	if sendingModeInfo != "" {
		leftSide = lipgloss.JoinHorizontal(lipgloss.Left, mode, port, connectionIndicator, sendingModeInfo, divider)
	} else {
		leftSide = lipgloss.JoinHorizontal(lipgloss.Left, mode, port, connectionIndicator, divider)
	}

	// Build right side with divider
	rightSide := lipgloss.JoinHorizontal(lipgloss.Left, connectionDetails, divider, time)

	// Calculate spacer
	leftWidth := lipgloss.Width(leftSide)
	rightWidth := lipgloss.Width(rightSide)
	spacerWidth := terminalWidth - leftWidth - rightWidth
	if spacerWidth < 1 {
		spacerWidth = 1
	}

	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	// Combine with background
	statusBarStyle := lipgloss.NewStyle().
		Foreground(colors.Text).
		Background(colors.Surface0).
		Width(terminalWidth)

	content := lipgloss.JoinHorizontal(lipgloss.Left, leftSide, spacer, rightSide)
	return statusBarStyle.Render(content)
}

// Keep the old View method for backward compatibility
func (sb *StatusBar) View(connected bool) string {
	return sb.ViewAsHeader(connected)
}
