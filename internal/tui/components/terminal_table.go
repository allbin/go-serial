package components

import (
	"fmt"
	"strings"

	"github.com/allbin/serial/internal/tui/colors"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewMode int

const (
	ViewModeFollow ViewMode = iota
	ViewModeVisual
)

type TerminalTable struct {
	table     table.Model
	formatter *DataFormatter
	viewMode  ViewMode
	rawData   []DataReceivedMsg
}

func NewTerminalTable(width, height int) *TerminalTable {
	// Ensure minimum dimensions for proper table initialization
	if width < 80 {
		width = 80
	}
	if height < 5 {
		height = 5
	}

	// Initial columns - will be updated by updateColumnsForDisplayMode
	columns := []table.Column{
		{Title: "Time", Width: 14},
		{Title: "↕", Width: 3},
		{Title: "Hex", Width: 30},
		{Title: "ASCII", Width: 20},
		{Title: "Bytes", Width: 6},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(false), // Start unfocused in follow mode
		table.WithHeight(height),
		table.WithWidth(width), // Use the corrected width
	)

	// Style the table
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colors.Subtext0).
		BorderBottom(true).
		Bold(true).
		Foreground(colors.Text)
	s.Selected = s.Selected.
		Foreground(colors.Text).
		Background(colors.Surface1).
		Bold(false)

	t.SetStyles(s)

	tt := &TerminalTable{
		table:     t,
		formatter: NewDataFormatter(true, true), // Default: show both hex and ASCII
		viewMode:  ViewModeFollow,               // Start in follow mode
		rawData:   make([]DataReceivedMsg, 0),
	}

	// Set initial column widths based on actual width
	tt.updateColumnsForDisplayMode(width)

	return tt
}

func (tt *TerminalTable) SetSize(width, height int) {
	// Update columns first, then table dimensions
	tt.updateColumnsForDisplayMode(width)
	tt.table.SetHeight(height)
	tt.table.SetWidth(width)

	// Update viewport after size changes
	tt.table.UpdateViewport()
}

func (tt *TerminalTable) updateColumnsForDisplayMode(width int) {
	displayMode := tt.formatter.GetDisplayMode()

	// Ensure minimum terminal width
	if width < 80 {
		width = 80
	}

	// Fixed column widths - keep these stable
	timeWidth := 14 // Increased for "15:04:05.000" format
	dirWidth := 3   // Just enough for the arrow
	bytesWidth := 6 // Enough for "Bytes" header and reasonable counts

	// Calculate remaining width for data columns
	// Account for borders and separators (roughly 8-10 chars)
	reservedWidth := timeWidth + dirWidth + bytesWidth + 10
	remainingWidth := width - reservedWidth
	if remainingWidth < 20 {
		remainingWidth = 20
	}

	var columns []table.Column

	if displayMode.ShowHex && displayMode.ShowASCII {
		// Both hex and ASCII columns
		// Give more space to hex since it's typically longer
		hexWidth := (remainingWidth * 7) / 10   // 70%
		asciiWidth := (remainingWidth * 3) / 10 // 30%

		// Enforce minimums
		if hexWidth < 20 {
			hexWidth = 20
		}
		if asciiWidth < 10 {
			asciiWidth = 10
		}

		columns = []table.Column{
			{Title: "Time", Width: timeWidth},
			{Title: "↕", Width: dirWidth},
			{Title: "Hex", Width: hexWidth},
			{Title: "ASCII", Width: asciiWidth},
			{Title: "Bytes", Width: bytesWidth},
		}
	} else if displayMode.ShowHex {
		// Hex only
		hexWidth := remainingWidth
		if hexWidth < 30 {
			hexWidth = 30
		}

		columns = []table.Column{
			{Title: "Time", Width: timeWidth},
			{Title: "↕", Width: dirWidth},
			{Title: "Hex", Width: hexWidth},
			{Title: "Bytes", Width: bytesWidth},
		}
	} else if displayMode.ShowASCII {
		// ASCII only
		asciiWidth := remainingWidth
		if asciiWidth < 20 {
			asciiWidth = 20
		}

		columns = []table.Column{
			{Title: "Time", Width: timeWidth},
			{Title: "↕", Width: dirWidth},
			{Title: "ASCII", Width: asciiWidth},
			{Title: "Bytes", Width: bytesWidth},
		}
	} else {
		// Neither hex nor ASCII
		dataWidth := remainingWidth
		if dataWidth < 25 {
			dataWidth = 25
		}

		columns = []table.Column{
			{Title: "Time", Width: timeWidth},
			{Title: "↕", Width: dirWidth},
			{Title: "Data", Width: dataWidth},
			{Title: "Bytes", Width: bytesWidth},
		}
	}

	tt.table.SetColumns(columns)
	// Update viewport after column changes
	tt.table.UpdateViewport()
}

func (tt *TerminalTable) AddMessage(msg DataReceivedMsg) {
	tt.rawData = append(tt.rawData, msg)

	// Debug: Log that we're adding a message
	// fmt.Printf("[DEBUG] Adding message: %d bytes, IsTX: %v, Status: %s\n", len(msg.Data), msg.IsTX, msg.Status)

	tt.refreshTable()

	// In follow mode, scroll to bottom
	if tt.viewMode == ViewModeFollow {
		tt.table.GotoBottom()
	}
}

func (tt *TerminalTable) UpdateMessage(rawData []DataReceivedMsg) {
	tt.rawData = rawData
	tt.refreshTable()

	// In follow mode, scroll to bottom
	if tt.viewMode == ViewModeFollow {
		tt.table.GotoBottom()
	}
}

func (tt *TerminalTable) refreshTable() {
	rows := make([]table.Row, len(tt.rawData))
	for i, msg := range tt.rawData {
		rows[i] = tt.formatMessageAsRow(msg)
	}

	tt.table.SetRows(rows)
	// Update viewport after row changes
	tt.table.UpdateViewport()
}

func (tt *TerminalTable) formatMessageAsRow(msg DataReceivedMsg) table.Row {
	// Format timestamp
	timestamp := msg.Timestamp.Format("15:04:05.000")

	// Format direction with arrows
	var direction string
	if msg.IsTX {
		direction = "↗"
	} else {
		direction = "↙"
	}

	// Format bytes count
	bytesStr := fmt.Sprintf("%d", len(msg.Data))

	// Format data based on current display mode and return appropriate row
	displayMode := tt.formatter.GetDisplayMode()

	if displayMode.ShowHex && displayMode.ShowASCII {
		// Both hex and ASCII columns
		hexStr := strings.ToUpper(fmt.Sprintf("% X", msg.Data))

		var asciiStr string
		for _, b := range msg.Data {
			if b >= 32 && b <= 126 {
				asciiStr += string(b)
			} else {
				asciiStr += "."
			}
		}

		return table.Row{timestamp, direction, hexStr, asciiStr, bytesStr}

	} else if displayMode.ShowHex {
		// Hex only
		hexStr := strings.ToUpper(fmt.Sprintf("% X", msg.Data))

		return table.Row{timestamp, direction, hexStr, bytesStr}

	} else if displayMode.ShowASCII {
		// ASCII only
		var asciiStr string
		for _, b := range msg.Data {
			if b >= 32 && b <= 126 {
				asciiStr += string(b)
			} else {
				asciiStr += "."
			}
		}

		return table.Row{timestamp, direction, asciiStr, bytesStr}

	} else {
		// Neither hex nor ASCII, show byte count info
		dataStr := fmt.Sprintf("%d bytes received", len(msg.Data))

		return table.Row{timestamp, direction, dataStr, bytesStr}
	}
}

func (tt *TerminalTable) Clear() {
	tt.rawData = make([]DataReceivedMsg, 0)
	tt.table.SetRows([]table.Row{})
}

func (tt *TerminalTable) ToggleHex() {
	tt.formatter.ToggleHex()
	// Update table structure to reflect new display mode
	tt.updateColumnsForDisplayMode(tt.table.Width())
	tt.refreshTable()
}

func (tt *TerminalTable) ToggleASCII() {
	tt.formatter.ToggleASCII()
	// Update table structure to reflect new display mode
	tt.updateColumnsForDisplayMode(tt.table.Width())
	tt.refreshTable()
}

func (tt *TerminalTable) GetDisplayMode() DisplayMode {
	return tt.formatter.GetDisplayMode()
}

func (tt *TerminalTable) GetViewMode() ViewMode {
	return tt.viewMode
}

func (tt *TerminalTable) SetViewMode(mode ViewMode) {
	tt.viewMode = mode
	if mode == ViewModeFollow {
		if len(tt.rawData) > 0 {
			tt.table.SetCursor(len(tt.rawData) - 1)
		}
		tt.table.GotoBottom()
		tt.table.Blur() // Unfocus in follow mode
	} else {
		tt.table.Focus() // Focus in visual mode for navigation
	}
	// Update viewport after mode changes
	tt.table.UpdateViewport()
}

func (tt *TerminalTable) RefreshDisplayWithRawData(rawData []DataReceivedMsg) {
	tt.rawData = rawData
	tt.refreshTable()
	if tt.viewMode == ViewModeFollow {
		tt.table.GotoBottom()
	}
}

func (tt *TerminalTable) Init() tea.Cmd {
	return nil
}

func (tt *TerminalTable) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Only allow table navigation in visual mode
	if tt.viewMode == ViewModeVisual {
		tt.table, cmd = tt.table.Update(msg)
	}

	return tt, cmd
}

func (tt *TerminalTable) View() string {
	return tt.table.View()
}

func (tt *TerminalTable) GetViewModeString() string {
	switch tt.viewMode {
	case ViewModeFollow:
		return "FOLLOW"
	case ViewModeVisual:
		return "VISUAL"
	default:
		return "FOLLOW"
	}
}
