package ui

import (
	"strings"

	"github.com/JamesYuuu/tick/internal/domain"
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
	// Left column: 7-day list as MM-DD.
	days := make([]string, 0, 7)
	for i := 0; i < 7; i++ {
		day := addDays(m.historyFrom, i)
		prefix := "  "
		if i == m.historyIndex {
			prefix = "> "
		}
		days = append(days, prefix+fmtMMDD(day))
	}
	left := strings.Join(days, "\n")

	// Right column: outcomes for selected day.
	rows := make([]string, 0, len(m.historyDone)+len(m.historyAbandoned)+len(m.historyActiveCreated)+1)

	for _, t := range m.historyDone {
		line := "[✓] " + t.Title
		// Delay for done tasks is determined by the actual completion day.
		// If DoneDay is missing, be conservative and do not mark delayed.
		if t.DoneDay != nil && t.DueDay.Before(*t.DoneDay) {
			line = m.styles.Delayed.Render(line)
		}
		rows = append(rows, line)
	}
	for _, t := range m.historyAbandoned {
		line := "[✗] " + t.Title
		// Delay for abandoned tasks is determined by the actual abandonment day.
		if t.AbandonedDay != nil && t.DueDay.Before(*t.AbandonedDay) {
			line = m.styles.Delayed.Render(line)
		}
		rows = append(rows, line)
	}

	// Overdue active tasks created on selected day.
	today := m.currentDay()
	for _, t := range m.historyActiveCreated {
		if t.Status != domain.StatusActive {
			continue
		}
		if !t.DueDay.Before(today) {
			continue
		}
		line := m.styles.Delayed.Render("[ ] " + t.Title)
		rows = append(rows, line)
	}
	if len(rows) == 0 {
		rows = append(rows, "(none)")
	}
	right := strings.Join(rows, "\n")

	// Divider: ASCII '|' sized to max lines.
	divider := verticalDivider(max(linesCount(left), linesCount(right)))

	cols := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(7).Render(left),
		" ",
		divider,
		" ",
		right,
	)
	return cols
}

func fmtMMDD(d domain.Day) string {
	// domain.Day is normalized to UTC midnight.
	return d.Time().Format("01-02")
}

func linesCount(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func verticalDivider(h int) string {
	if h <= 0 {
		return ""
	}
	lines := make([]string, 0, h)
	for i := 0; i < h; i++ {
		lines = append(lines, "|")
	}
	return strings.Join(lines, "\n")
}
