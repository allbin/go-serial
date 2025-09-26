package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/allbin/serial/internal/tui/colors"
	"github.com/charmbracelet/lipgloss"
)

type DataReceivedMsg struct {
	Timestamp time.Time
	Data      []byte
	IsTX      bool
	Status    string // For TX messages: "PENDING", "WRITTEN", "ERROR", empty for RX
}

type DisplayMode struct {
	ShowHex   bool
	ShowASCII bool
}

type DataFormatter struct {
	mode DisplayMode
}

func NewDataFormatter(showHex, showASCII bool) *DataFormatter {
	return &DataFormatter{
		mode: DisplayMode{
			ShowHex:   showHex,
			ShowASCII: showASCII,
		},
	}
}

func (df *DataFormatter) SetDisplayMode(showHex, showASCII bool) {
	df.mode.ShowHex = showHex
	df.mode.ShowASCII = showASCII
}

func (df *DataFormatter) GetDisplayMode() DisplayMode {
	return df.mode
}

func (df *DataFormatter) FormatMessage(msg DataReceivedMsg) string {
	timestamp := msg.Timestamp.Format("15:04:05.000")

	// Create styled TX/RX indicators with arrows and status
	var indicator string
	if msg.IsTX {
		// TX with up-right arrow and status-based coloring
		var txColor lipgloss.Color
		var statusText string

		switch msg.Status {
		case "PENDING":
			txColor = colors.Yellow
			statusText = "TX ○"
		case "TRANSMITTING":
			txColor = colors.Blue
			statusText = "TX ⏸" // Pause symbol for blocked/transmitting
		case "WRITTEN":
			txColor = colors.Green
			statusText = "TX ✓"
		case "ERROR":
			txColor = colors.Red
			statusText = "TX ✗"
		default:
			txColor = colors.Peach
			statusText = "TX"
		}

		indicator = lipgloss.NewStyle().
			Foreground(txColor).
			Bold(true).
			Render("↗ " + statusText)
	} else {
		// RX with down-left arrow and blue color
		indicator = lipgloss.NewStyle().
			Foreground(colors.Sky).
			Bold(true).
			Render("↙ RX")
	}

	var parts []string

	if df.mode.ShowHex {
		hexStr := fmt.Sprintf("% X", msg.Data)
		parts = append(parts, fmt.Sprintf("HEX: %s", hexStr))
	}

	if df.mode.ShowASCII {
		asciiStr := ""
		for _, b := range msg.Data {
			if b >= 32 && b <= 126 {
				// Only include printable ASCII characters
				asciiStr += string(b)
			} else {
				// Replace non-printable characters with dots
				asciiStr += "."
			}
		}
		// Ensure the ASCII string doesn't contain any terminal control sequences
		parts = append(parts, fmt.Sprintf("ASCII: %s", asciiStr))
	}

	// If both are disabled, show raw bytes count
	if !df.mode.ShowHex && !df.mode.ShowASCII {
		parts = append(parts, fmt.Sprintf("BYTES: %d", len(msg.Data)))
	}

	// Style timestamp
	timestampStyled := lipgloss.NewStyle().
		Foreground(colors.Subtext0).
		Render(fmt.Sprintf("[%s]", timestamp))

	return fmt.Sprintf("%s %s: %s", timestampStyled, indicator, strings.Join(parts, "  "))
}

func (df *DataFormatter) FormatMessages(messages []DataReceivedMsg) []string {
	formatted := make([]string, len(messages))
	for i, msg := range messages {
		formatted[i] = df.FormatMessage(msg)
	}
	return formatted
}

func (df *DataFormatter) ToggleHex() {
	df.mode.ShowHex = !df.mode.ShowHex
}

func (df *DataFormatter) ToggleASCII() {
	df.mode.ShowASCII = !df.mode.ShowASCII
}
