package ui

func renderToday(m Model) string {
	body := m.styles.Body.Render("Today\n\n(placeholder)")
	return m.frame("Today", body)
}

func renderUpcoming(m Model) string {
	body := m.styles.Body.Render("Upcoming\n\n(placeholder)")
	return m.frame("Upcoming", body)
}

func renderHistory(m Model) string {
	body := m.styles.Body.Render("History\n\n(placeholder)")
	return m.frame("History", body)
}
