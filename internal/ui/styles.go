package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	AppTitle lipgloss.Style
	Tab      lipgloss.Style
	TabOn    lipgloss.Style
	Help     lipgloss.Style
	Body     lipgloss.Style
}

func defaultStyles() styles {
	base := lipgloss.NewStyle().Padding(0, 1)
	return styles{
		AppTitle: lipgloss.NewStyle().Bold(true),
		Tab:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		TabOn:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
		Help:     lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		Body:     base,
	}
}

func (m Model) frame(title string, body string) string {
	header := m.header(title)
	help := m.help()

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")
	b.WriteString(body)
	b.WriteString("\n\n")
	b.WriteString(help)
	b.WriteString("\n")
	return b.String()
}

func (m Model) header(active string) string {
	left := m.styles.AppTitle.Render("tuitodo")
	tabs := []string{
		m.tab("Today", active == "Today"),
		m.tab("Upcoming", active == "Upcoming"),
		m.tab("History", active == "History"),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  "+strings.Join(tabs, "  "))
}

func (m Model) tab(name string, on bool) string {
	if on {
		return m.styles.TabOn.Render("[" + name + "]")
	}
	return m.styles.Tab.Render(name)
}

func (m Model) help() string {
	return m.styles.Help.Render("1:Today  2:Upcoming  3:History  q:Quit")
}
