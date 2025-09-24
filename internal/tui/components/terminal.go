package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Terminal struct {
	viewport  viewport.Model
	formatter *DataFormatter
	data      []string
}

func NewTerminal(width, height int) *Terminal {
	vp := viewport.New(width, height)
	return &Terminal{
		viewport:  vp,
		formatter: NewDataFormatter(true, true), // Default: show both hex and ASCII
		data:      make([]string, 0),
	}
}

func (t *Terminal) SetSize(width, height int) {
	t.viewport.Width = width
	t.viewport.Height = height
}

func (t *Terminal) GetViewport() viewport.Model {
	return t.viewport
}

func (t *Terminal) AddMessage(msg DataReceivedMsg) {
	formattedMsg := t.formatter.FormatMessage(msg)
	t.data = append(t.data, formattedMsg)

	// Set content and ensure viewport scrolls to show the latest message
	content := strings.Join(t.data, "\n")
	t.viewport.SetContent(content)

	// Force viewport to bottom to show the latest message
	// Even if content is shorter than viewport height
	if len(t.data) > 0 {
		t.viewport.GotoBottom()
	}
}

func (t *Terminal) UpdateMessage(rawData []DataReceivedMsg) {
	// Refresh the entire display with updated raw data
	// This ensures proper ordering and formatting
	t.data = t.formatter.FormatMessages(rawData)
	t.viewport.SetContent(strings.Join(t.data, "\n"))
	t.viewport.GotoBottom()
}

func (t *Terminal) AddFormattedMessage(msg string) {
	t.data = append(t.data, msg)
	t.viewport.SetContent(strings.Join(t.data, "\n"))
	t.viewport.GotoBottom()
}

func (t *Terminal) RefreshDisplayWithRawData(rawData []DataReceivedMsg) {
	t.data = t.formatter.FormatMessages(rawData)
	t.viewport.SetContent(strings.Join(t.data, "\n"))
	t.viewport.GotoBottom()
}

func (t *Terminal) Clear() {
	t.data = make([]string, 0)
	t.viewport.SetContent("")
}

func (t *Terminal) ToggleHex() {
	t.formatter.ToggleHex()
}

func (t *Terminal) ToggleASCII() {
	t.formatter.ToggleASCII()
}

func (t *Terminal) GetDisplayMode() DisplayMode {
	return t.formatter.GetDisplayMode()
}

func (t *Terminal) Update(msg tea.Msg) (viewport.Model, tea.Cmd) {
	// Only pass certain message types to viewport to prevent it from consuming our key bindings
	switch msg.(type) {
	case tea.WindowSizeMsg:
		return t.viewport.Update(msg)
	default:
		// Don't pass other message types (like KeyMsg) to viewport
		return t.viewport, nil
	}
}

func (t *Terminal) View() string {
	return t.viewport.View()
}
