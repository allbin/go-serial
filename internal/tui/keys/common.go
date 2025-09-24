package keys

import "github.com/charmbracelet/bubbles/key"

// Common key bindings used across TUI commands
type CommonKeys struct {
	Quit       key.Binding
	Help       key.Binding
	InsertMode key.Binding
	Escape     key.Binding
}

func NewCommonKeys() CommonKeys {
	return CommonKeys{
		Quit: key.NewBinding(
			key.WithKeys("q", "Q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		InsertMode: key.NewBinding(
			key.WithKeys("i", "I"),
			key.WithHelp("i", "insert mode"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "normal mode"),
		),
	}
}

// Terminal-specific key bindings for commands that display data
type TerminalKeys struct {
	CommonKeys
	Clear       key.Binding
	ToggleHex   key.Binding
	ToggleASCII key.Binding
}

func NewTerminalKeys() TerminalKeys {
	return TerminalKeys{
		CommonKeys: NewCommonKeys(),
		Clear: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clear buffer"),
		),
		ToggleHex: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "toggle hex"),
		),
		ToggleASCII: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle ascii"),
		),
	}
}

func (k TerminalKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.InsertMode, k.Clear, k.Quit}
}

func (k TerminalKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.InsertMode, k.Escape, k.Clear, k.ToggleHex, k.ToggleASCII},
		{k.Help, k.Quit},
	}
}
