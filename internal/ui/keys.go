package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Today    key.Binding
	Upcoming key.Binding
	History  key.Binding
	Add      key.Binding
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
		Today: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "today"),
		),
		Upcoming: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "upcoming"),
		),
		History: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "history"),
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
		HistoryLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("left/h", "-1 day window"),
		),
		HistoryRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("right/l", "+1 day window"),
		),
	}
}
