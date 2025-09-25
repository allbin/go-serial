package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mdjarv/serial/internal/tui/colors"
	"github.com/mdjarv/serial/internal/tui/styles"
)

type SendingMode int

const (
	SendingModeASCII SendingMode = iota
	SendingModeHex
)

func (s SendingMode) String() string {
	switch s {
	case SendingModeASCII:
		return "ASCII"
	case SendingModeHex:
		return "HEX"
	default:
		return "ASCII"
	}
}

type Input struct {
	textInput     textinput.Model
	sendingMode   SendingMode
	history       []string
	historyIndex  int
	currentInput  string // Store current input when navigating history
	terminalWidth int    // Store terminal width for consistent sizing
}

func NewInput(placeholder string) *Input {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 256
	ti.Prompt = "" // We handle prompt styling separately
	ti.Focus()

	// Default to hex mode with Neocortec test message pre-populated
	ti.SetValue("0206000300000099")

	return &Input{
		textInput:    ti,
		sendingMode:  SendingModeHex,
		history:      make([]string, 0),
		historyIndex: -1,
		currentInput: "",
	}
}

func (i *Input) SetWidth(width int) {
	// Store the terminal width for reference
	i.terminalWidth = width
	// Account for: border(2) + padding(2) + prompt(1) + space(1) = 6 characters
	usableWidth := width - 6
	if usableWidth < 20 {
		usableWidth = 20 // Minimum usable width
	}
	i.textInput.Width = usableWidth
}

func (i *Input) Focus() {
	i.textInput.Focus()
}

func (i *Input) Blur() {
	i.textInput.Blur()
}

func (i *Input) Value() string {
	return i.textInput.Value()
}

func (i *Input) SetValue(value string) {
	i.textInput.SetValue(value)
}

func (i *Input) ToggleSendingMode() {
	switch i.sendingMode {
	case SendingModeASCII:
		i.sendingMode = SendingModeHex
		i.textInput.Placeholder = "Enter hex (e.g. 48656C6C6F or 48 65 6C 6C 6F)..."
	case SendingModeHex:
		i.sendingMode = SendingModeASCII
		i.textInput.Placeholder = "Type message and press Enter to send..."
	}
}

func (i *Input) GetSendingMode() SendingMode {
	return i.sendingMode
}

func (i *Input) Update(msg tea.Msg) (*Input, tea.Cmd) {
	var cmd tea.Cmd
	i.textInput, cmd = i.textInput.Update(msg)
	return i, cmd
}

func (i *Input) View() string {
	sendModeIndicator := lipgloss.NewStyle().
		Foreground(colors.Overlay0).
		Render(fmt.Sprintf("[%s] ", i.sendingMode.String()))

	inputView := styles.InputStyle.Render(i.textInput.View())

	return lipgloss.JoinHorizontal(lipgloss.Left, sendModeIndicator, inputView)
}

func (i *Input) ViewWithMode(inputMode string, isInsertMode bool) string {
	// Create consistent styling for both modes
	var promptStyle lipgloss.Style
	var inputContent string

	// Clean prompt symbols with foreground colors only
	var promptSymbol string
	if i.sendingMode == SendingModeHex {
		promptSymbol = "#"
		promptStyle = lipgloss.NewStyle().
			Foreground(colors.Yellow).
			Bold(true)
	} else {
		promptSymbol = ">"
		promptStyle = lipgloss.NewStyle().
			Foreground(colors.Green).
			Bold(true)
	}

	styledPrompt := promptStyle.Render(promptSymbol)

	if isInsertMode {
		// Insert mode: show input field (Tab hint moved to status bar)
		inputField := i.textInput.View()
		inputContent = lipgloss.JoinHorizontal(lipgloss.Left, styledPrompt, " ", inputField)
	} else {
		// Normal mode: show instruction
		instruction := lipgloss.NewStyle().
			Foreground(colors.Overlay0).
			Render("Press 'i' to enter insert mode")
		inputContent = lipgloss.JoinHorizontal(lipgloss.Left, styledPrompt, " ", instruction)
	}

	// Apply border styling with width adjusted for border and padding
	// RoundedBorder adds 2 characters (left + right), padding adds 2 characters (0,1 on each side)
	// So we need to account for 4 characters total
	adjustedWidth := i.terminalWidth - 4
	if adjustedWidth < 10 {
		adjustedWidth = 10
	}

	// Create input style with highlighting when in insert mode
	inputStyle := styles.InputStyle.Copy().
		Width(adjustedWidth).
		AlignHorizontal(lipgloss.Left)

	if isInsertMode {
		// Highlight the input field in insert mode with a green border (matching INSERT mode indicator)
		inputStyle = inputStyle.
			BorderForeground(colors.Green)
	}

	return inputStyle.Render(inputContent)
}

// AddToHistory adds a command to the history if it's not empty or a duplicate
func (i *Input) AddToHistory(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	// Don't add if it's the same as the last command
	if len(i.history) > 0 && i.history[len(i.history)-1] == command {
		return
	}

	i.history = append(i.history, command)

	// Keep only last 100 commands
	if len(i.history) > 100 {
		i.history = i.history[1:]
	}

	// Reset history index
	i.historyIndex = -1
	i.currentInput = ""
}

// NavigateHistoryUp moves up in command history
func (i *Input) NavigateHistoryUp() {
	if len(i.history) == 0 {
		return
	}

	// First time navigating: save current input
	if i.historyIndex == -1 {
		i.currentInput = i.textInput.Value()
		i.historyIndex = len(i.history) - 1
	} else if i.historyIndex > 0 {
		i.historyIndex--
	}

	i.textInput.SetValue(i.history[i.historyIndex])
}

// NavigateHistoryDown moves down in command history
func (i *Input) NavigateHistoryDown() {
	if len(i.history) == 0 || i.historyIndex == -1 {
		return
	}

	if i.historyIndex < len(i.history)-1 {
		i.historyIndex++
		i.textInput.SetValue(i.history[i.historyIndex])
	} else {
		// Back to current input
		i.historyIndex = -1
		i.textInput.SetValue(i.currentInput)
		i.currentInput = ""
	}
}
