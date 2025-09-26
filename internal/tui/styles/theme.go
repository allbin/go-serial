package styles

import (
	"github.com/allbin/serial/internal/tui/colors"
	"github.com/charmbracelet/lipgloss"
)

var (
	// Header styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Mauve).
			Background(colors.Surface0).
			Padding(0, 1)

	// Status styles
	StatusConnectedStyle = lipgloss.NewStyle().
				Foreground(colors.Green).
				Bold(true)

	StatusDisconnectedStyle = lipgloss.NewStyle().
				Foreground(colors.Red).
				Bold(true)

	StatusConnectingStyle = lipgloss.NewStyle().
				Foreground(colors.Yellow).
				Bold(true)

	// Content area styles
	ContentBorderStyle = lipgloss.NewStyle().
				BorderTop(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(colors.Surface1)

	// Input styles
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colors.Surface2).
			Padding(0, 1)

	// Error styles
	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Red).
			Align(lipgloss.Center)

	// Info styles
	InfoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colors.Mauve).
			Align(lipgloss.Center)
)

type StatusType int

const (
	StatusConnected StatusType = iota
	StatusDisconnected
	StatusConnecting
	StatusError
)

func GetStatusStyle(status StatusType) lipgloss.Style {
	switch status {
	case StatusConnected:
		return StatusConnectedStyle
	case StatusDisconnected:
		return StatusDisconnectedStyle
	case StatusConnecting:
		return StatusConnectingStyle
	case StatusError:
		return StatusDisconnectedStyle
	default:
		return StatusDisconnectedStyle
	}
}
