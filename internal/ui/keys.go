package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	NextView key.Binding
	Add      key.Binding
	Edit     key.Binding
	Delete   key.Binding
	Done     key.Binding
	Abandon  key.Binding
	Postpone key.Binding
	Quit     key.Binding

	HistoryUp    key.Binding
	HistoryDown  key.Binding
	HistoryLeft  key.Binding
	HistoryRight key.Binding
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
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d", "delete"),
			key.WithHelp("d", "delete"),
		),
		Done: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "done"),
		),
		Abandon: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "abandon"),
		),
		Postpone: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "postpone"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		HistoryUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("up/k", "scroll up"),
		),
		HistoryDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("down/j", "scroll down"),
		),
		HistoryLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("left/h", "prev day"),
		),
		HistoryRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("right/l", "next day"),
		),
	}
}
