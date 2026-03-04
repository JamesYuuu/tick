package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type styles struct {
	AppTitle lipgloss.Style
	Tab      lipgloss.Style
	TabOn    lipgloss.Style
	Help     lipgloss.Style
	Body     lipgloss.Style
	Status   lipgloss.Style
	Delayed  lipgloss.Style
}

func defaultStyles() styles {
	base := lipgloss.NewStyle().Padding(0, 1)
	return styles{
		AppTitle: lipgloss.NewStyle().Bold(true),
		Tab:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		TabOn:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
		Help:     lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		Body:     base,
		Status:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Delayed:  lipgloss.NewStyle().Foreground(lipgloss.Color("9")),
	}
}

func (m Model) frame(title string, body string) string {
	header := m.header(title)
	help := m.help()
	status := ""
	if m.statusMsg != "" {
		status = m.styles.Status.Render(m.statusMsg)
	}

	historyFooter := ""
	if m.view == viewHistory {
		historyFooter = "DoneDelayedRatio: " + fmtRatio(m.historyStats.DoneDelayedRatio) +
			"  AbandonedDelayedRatio: " + fmtRatio(m.historyStats.AbandonedDelayedRatio)
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n\n")
	b.WriteString(body)
	b.WriteString("\n\n")
	if status != "" {
		b.WriteString(status)
		b.WriteString("\n")
	}
	if status == "" && historyFooter != "" {
		b.WriteString(m.styles.Status.Render(historyFooter))
		b.WriteString("\n")
	}
	b.WriteString(help)
	b.WriteString("\n")
	return b.String()
}

func fmtRatio(f float64) string {
	// Keep it simple for TUI.
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
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
	base := "1:Today  2:Upcoming  3:History  q:Quit"
	suffix := ""
	switch m.view {
	case viewToday:
		suffix = "  a:Add  x:Done  d:Abandon  p:+1 day"
	case viewHistory:
		suffix = "  up/k:prev day  down/j:next day  left/h:-1 day window  right/l:+1 day window"
	}
	return m.styles.Help.Render(base + suffix)
}
