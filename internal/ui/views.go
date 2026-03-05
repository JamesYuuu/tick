package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func renderToday(m Model) string {
	body := renderTodayBody(m)
	return m.frame("Today", body)
}

func renderTodayBody(m Model) string {
	if m.adding {
		return "Add task\n\n" + m.addInput.View()
	}
	if len(m.todayList.Items()) == 0 {
		return "Nothing due today."
	}
	return m.todayList.View()
}

func renderUpcoming(m Model) string {
	body := renderUpcomingBody(m)
	return m.frame("Upcoming", body)
}

func renderUpcomingBody(m Model) string {
	if len(m.upcomingList.Items()) == 0 {
		return "No upcoming tasks."
	}
	return m.upcomingList.View()
}

func renderHistory(m Model) string {
	body := renderHistoryBody(m)
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
	if len(m.historyDone) == 0 {
		doneLines = append(doneLines, "(none)")
	}

	abLines := make([]string, 0, len(m.historyAbandoned)+1)
	abLines = append(abLines, "Abandoned")
	for _, t := range m.historyAbandoned {
		abLines = append(abLines, "- "+t.Title)
	}
	if len(m.historyAbandoned) == 0 {
		abLines = append(abLines, "(none)")
	}

	right := strings.Join(append(doneLines, "", strings.Join(abLines, "\n")), "\n")

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(12).Render(left),
		"  ",
		right,
	)
	return cols
}
