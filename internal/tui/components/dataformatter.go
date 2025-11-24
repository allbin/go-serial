package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/allbin/go-serial/internal/tui/colors"
	"github.com/charmbracelet/lipgloss"
)

type DataReceivedMsg struct {
	Timestamp    time.Time
	Data         []byte
	IsTX         bool
	Status       string     // For TX messages: "PENDING", "WRITTEN", "ERROR", empty for RX
	Sequence     int64      // Unique sequence number for updating messages in place
	EnqueuedTime *time.Time // When message was queued for sending (TX only)
	WrittenTime  *time.Time // When message was actually written (TX only)
}

type DisplayMode struct {
	ShowHex   bool
	ShowASCII bool
}

type FormatOptions struct {
	NoTimestamps bool // Hide timestamps
	NoIndicators bool // Hide RX/TX indicators
}

type DataFormatter struct {
	mode       DisplayMode
	options    FormatOptions
	lineBuffer []byte // Buffer for accumulating partial lines in ASCII mode
}

func NewDataFormatter(showHex, showASCII bool) *DataFormatter {
	return &DataFormatter{
		mode: DisplayMode{
			ShowHex:   showHex,
			ShowASCII: showASCII,
		},
		lineBuffer: make([]byte, 0, 256),
	}
}

func (df *DataFormatter) SetDisplayMode(showHex, showASCII bool) {
	df.mode.ShowHex = showHex
	df.mode.ShowASCII = showASCII
}

func (df *DataFormatter) GetDisplayMode() DisplayMode {
	return df.mode
}

func (df *DataFormatter) SetFormatOptions(noTimestamps, noIndicators bool) {
	df.options.NoTimestamps = noTimestamps
	df.options.NoIndicators = noIndicators
}

func (df *DataFormatter) FormatMessage(msg DataReceivedMsg) []string {
	// For TX messages or HEX-only mode, show each chunk immediately (original behavior)
	if msg.IsTX || (df.mode.ShowHex && !df.mode.ShowASCII) {
		line := df.formatSingleChunk(msg, msg.Data)
		if line != "" {
			return []string{line}
		}
		return []string{}
	}

	// For RX messages with ASCII mode enabled, buffer lines
	if df.mode.ShowASCII {
		return df.formatBufferedLines(msg)
	}

	// Fallback for other cases
	line := df.formatSingleChunk(msg, msg.Data)
	if line != "" {
		return []string{line}
	}
	return []string{}
}

// formatSingleChunk formats a single data chunk without line buffering
func (df *DataFormatter) formatSingleChunk(msg DataReceivedMsg, data []byte) string {
	var parts []string

	// Add timestamp if enabled
	var timestampStyled string
	if !df.options.NoTimestamps {
		timestamp := msg.Timestamp.Format("15:04:05.000")
		timestampStyled = lipgloss.NewStyle().
			Foreground(colors.Subtext0).
			Render(fmt.Sprintf("[%s]", timestamp))
	}

	// Add indicator if enabled
	var indicator string
	if !df.options.NoIndicators {
		indicator = df.getIndicator(msg)
	}

	// Format data with visual styling (no prefixes, just colors)
	if df.mode.ShowHex {
		hexStr := fmt.Sprintf("% X", data)
		hexPart := lipgloss.NewStyle().
			Foreground(colors.Peach).
			Render(hexStr)
		parts = append(parts, hexPart)
	}

	if df.mode.ShowASCII {
		asciiStr := df.bytesToASCII(data)
		// ASCII in default color (no styling needed)
		parts = append(parts, asciiStr)
	}

	// If both are disabled, show raw bytes count
	if !df.mode.ShowHex && !df.mode.ShowASCII {
		parts = append(parts, fmt.Sprintf("%d bytes", len(data)))
	}

	// Assemble the final output based on options
	var result string
	// Use a styled divider between HEX and ASCII when both are shown
	divider := lipgloss.NewStyle().
		Foreground(colors.Overlay0).
		Render(" │ ")
	dataStr := strings.Join(parts, divider)

	if df.options.NoTimestamps && df.options.NoIndicators {
		// Raw mode: just the data
		result = dataStr
	} else if df.options.NoTimestamps {
		// No timestamp, but show indicator
		result = fmt.Sprintf("%s: %s", indicator, dataStr)
	} else if df.options.NoIndicators {
		// Timestamp only, no indicator
		result = fmt.Sprintf("%s %s", timestampStyled, dataStr)
	} else {
		// Full format
		result = fmt.Sprintf("%s %s: %s", timestampStyled, indicator, dataStr)
	}

	return result
}

// formatBufferedLines handles line buffering for ASCII mode
func (df *DataFormatter) formatBufferedLines(msg DataReceivedMsg) []string {
	df.lineBuffer = append(df.lineBuffer, msg.Data...)

	var lines []string

	// Process complete lines from buffer
	for {
		// Find line ending (\n or \r\n)
		idx := -1
		lineEndLen := 0

		for i := 0; i < len(df.lineBuffer); i++ {
			if df.lineBuffer[i] == '\n' {
				idx = i
				lineEndLen = 1
				// Check for \r\n
				if i > 0 && df.lineBuffer[i-1] == '\r' {
					idx = i - 1
					lineEndLen = 2
				}
				break
			}
		}

		if idx == -1 {
			// No complete line yet, keep buffering
			break
		}

		// Extract complete line (without line ending)
		lineData := make([]byte, idx)
		copy(lineData, df.lineBuffer[:idx])

		// Format the line
		line := df.formatSingleChunk(msg, lineData)
		if line != "" {
			lines = append(lines, line)
		}

		// Remove processed line from buffer
		df.lineBuffer = df.lineBuffer[idx+lineEndLen:]
	}

	return lines
}

// getIndicator creates the styled TX/RX indicator
func (df *DataFormatter) getIndicator(msg DataReceivedMsg) string {
	if msg.IsTX {
		// TX with up-right arrow and status-based coloring
		var txColor lipgloss.Color
		var statusText string

		switch msg.Status {
		case "PENDING":
			txColor = colors.Yellow
			statusText = "TX [ENQUEUED]"
		case "WRITTEN":
			txColor = colors.Green
			statusText = "TX [SENT"
			// Show timing delta if we have both enqueued and written times
			if msg.EnqueuedTime != nil && msg.WrittenTime != nil {
				delta := msg.WrittenTime.Sub(*msg.EnqueuedTime)
				statusText += fmt.Sprintf(" +%dms", delta.Milliseconds())
			}
			statusText += "]"
		case "TIMEOUT":
			txColor = colors.Peach // Orange/peach for timeout
			statusText = "TX [TIMEOUT - MAY STILL SEND]"
		case "ERROR":
			txColor = colors.Red
			statusText = "TX [ERROR]"
		default:
			txColor = colors.Peach
			statusText = "TX"
		}

		return lipgloss.NewStyle().
			Foreground(txColor).
			Bold(true).
			Render("↗ " + statusText)
	}

	// RX with down-left arrow and blue color
	return lipgloss.NewStyle().
		Foreground(colors.Sky).
		Bold(true).
		Render("↙ RX")
}

// bytesToASCII converts bytes to ASCII string with non-printable chars as dots
func (df *DataFormatter) bytesToASCII(data []byte) string {
	var result strings.Builder
	for _, b := range data {
		if b >= 32 && b <= 126 {
			result.WriteByte(b)
		} else if b == '\t' {
			result.WriteByte('\t')
		} else if b == '\r' || b == '\n' {
			// Skip line endings in ASCII output
			continue
		} else {
			result.WriteByte('.')
		}
	}
	return result.String()
}

func (df *DataFormatter) FormatMessages(messages []DataReceivedMsg) []string {
	var formatted []string
	for _, msg := range messages {
		lines := df.FormatMessage(msg)
		formatted = append(formatted, lines...)
	}
	return formatted
}

// FlushBuffer forces any buffered data to be output
func (df *DataFormatter) FlushBuffer(timestamp time.Time) []string {
	if len(df.lineBuffer) == 0 {
		return []string{}
	}

	// Create a dummy message for the buffered data
	msg := DataReceivedMsg{
		Timestamp: timestamp,
		Data:      df.lineBuffer,
		IsTX:      false,
	}

	line := df.formatSingleChunk(msg, df.lineBuffer)
	df.lineBuffer = df.lineBuffer[:0] // Clear buffer

	if line != "" {
		return []string{line}
	}
	return []string{}
}

// ClearBuffer clears any buffered data without outputting it
func (df *DataFormatter) ClearBuffer() {
	df.lineBuffer = df.lineBuffer[:0]
}

func (df *DataFormatter) ToggleHex() {
	df.mode.ShowHex = !df.mode.ShowHex
}

func (df *DataFormatter) ToggleASCII() {
	df.mode.ShowASCII = !df.mode.ShowASCII
}
