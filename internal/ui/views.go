package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderToday(m Model) string {
	var body string
	if m.adding {
		body = m.styles.Body.Render("Add task\n\n" + m.addInput.View())
	} else {
		body = m.styles.Body.Render(m.todayList.View())
	}
	return m.frame("Today", body)
}

func renderUpcoming(m Model) string {
	body := m.styles.Body.Render(m.upcomingList.View())
	return m.frame("Upcoming", body)
}

func renderHistory(m Model) string {
	body := m.styles.Body.Render(renderHistoryBody(m))
	return m.frame("History", body)
}

func renderHistoryBody(m Model) string {
	days := make([]string, 0, 7)
	for i := 0; i < 7; i++ {
		day := addDays(m.historyFrom, i)
		prefix := "  "
		if i == m.historyIndex {
			prefix = "> "
		}
		days = append(days, prefix+day.String())
	}
	left := strings.Join(days, "\n")

	doneLines := make([]string, 0, len(m.historyDone)+1)
	doneLines = append(doneLines, "Done")
	for _, t := range m.historyDone {
		doneLines = append(doneLines, "- "+t.Title)
	}

	abLines := make([]string, 0, len(m.historyAbandoned)+1)
	abLines = append(abLines, "Abandoned")
	for _, t := range m.historyAbandoned {
		abLines = append(abLines, "- "+t.Title)
	}

	right := strings.Join(append(doneLines, "", strings.Join(abLines, "\n")), "\n")

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(12).Render(left),
		"  ",
		right,
	)
	return cols
}
