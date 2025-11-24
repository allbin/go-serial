package keys

import "github.com/charmbracelet/bubbles/key"

// ConnectKeys includes terminal keys plus send/input functionality
type ConnectKeys struct {
	TerminalKeys
	Enter          key.Binding
	Send           key.Binding
	ToggleSendMode key.Binding
	Up             key.Binding
	Down           key.Binding
	VisualMode     key.Binding
	GotoTop        key.Binding
	GotoBottom     key.Binding
}

func NewConnectKeys() ConnectKeys {
	return ConnectKeys{
		TerminalKeys: NewTerminalKeys(),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "send message"),
		),
		Send: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "send message"),
		),
		ToggleSendMode: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "toggle send mode"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		VisualMode: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "visual mode"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "goto top"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "goto bottom"),
		),
	}
}

func (k ConnectKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.InsertMode, k.VisualMode, k.Enter, k.Quit}
}

func (k ConnectKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.InsertMode, k.VisualMode, k.Escape, k.Clear},
		{k.ToggleHex, k.ToggleASCII, k.ToggleTimestamps, k.ToggleIndicators},
		{k.GotoTop, k.GotoBottom, k.Up, k.Down},
		{k.Enter, k.Help, k.Quit},
	}
}
