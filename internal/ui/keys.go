package ui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Today    key.Binding
	Upcoming key.Binding
	History  key.Binding
	Quit     key.Binding
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
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}
