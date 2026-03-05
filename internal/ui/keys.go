package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	NextView key.Binding
	Add      key.Binding
	Done     key.Binding
	Abandon  key.Binding
	Postpone key.Binding
	Quit     key.Binding

	HistoryUp   key.Binding
	HistoryDown key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		NextView: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next view"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add"),
		),
		Done: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "done"),
		),
		Abandon: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "abandon"),
		),
		Postpone: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "+1 day"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		HistoryUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "prev day"),
		),
		HistoryDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "next day"),
		),
	}
}
