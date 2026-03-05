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
	Sheet    lipgloss.Style
	RowSel   lipgloss.Style
	RowSelDl lipgloss.Style
	Status   lipgloss.Style
	Delayed  lipgloss.Style
}

const maxContentWidth = 96

func contentWidth(windowWidth int) int {
	if windowWidth <= 0 {
		return 0
	}
	if windowWidth > maxContentWidth {
		return maxContentWidth
	}
	return windowWidth
}

// Sheet styles: border (1+1) + padding (1+1).
const sheetHorizMargin = 4

// Sheet styles: top + bottom border.
const sheetVertMargin = 2

func sheetInnerWidth(windowWidth int) int {
	w := contentWidth(windowWidth) - sheetHorizMargin
	if w < 0 {
		return 0
	}
	return w
}

func separatorLine(windowWidth int) string {
	w := contentWidth(windowWidth)
	if w <= 0 {
		return ""
	}
	return strings.Repeat("-", w)
}

// padLeftToWidth prefixes each line with spaces so the rendered block is
// centered within the given window width, capped by maxContentWidth.
//
// ASCII-only: assumes spaces are width 1 and doesn't account for rune widths.
func padLeftToWidth(s string, windowWidth int) string {
	cw := contentWidth(windowWidth)
	pad := (windowWidth - cw) / 2
	if pad <= 0 || s == "" {
		return s
	}

	prefix := strings.Repeat(" ", pad)
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

// forceHeight ensures the string has exactly h lines by adding or trimming
// trailing newlines.
func forceHeight(s string, h int) string {
	if h <= 0 {
		return s
	}
	trimmed := strings.TrimRight(s, "\n")
	parts := []string{""}
	if trimmed != "" {
		parts = strings.Split(trimmed, "\n")
	}
	if len(parts) > h {
		parts = parts[:h]
	}
	for len(parts) < h {
		// Use a single space so the line survives TrimRight("\n") callers.
		parts = append(parts, " ")
	}
	return strings.Join(parts, "\n")
}

func defaultStyles() styles {
	sheetBorder := lipgloss.Border{
		Top:         "-",
		Bottom:      "-",
		Left:        "|",
		Right:       "|",
		TopLeft:     "+",
		TopRight:    "+",
		BottomLeft:  "+",
		BottomRight: "+",
	}

	return styles{
		AppTitle: lipgloss.NewStyle().Bold(true),
		Tab:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		TabOn:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")),
		Help:     lipgloss.NewStyle().Foreground(lipgloss.Color("242")),
		Sheet:    lipgloss.NewStyle().Padding(0, 1).Border(sheetBorder).BorderForeground(lipgloss.Color("240")),
		RowSel:   lipgloss.NewStyle().Background(lipgloss.Color("254")).Foreground(lipgloss.Color("236")),
		RowSelDl: lipgloss.NewStyle().Background(lipgloss.Color("254")).Foreground(lipgloss.Color("1")),
		Status:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Delayed:  lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	}
}

func (m Model) frame(title string, body string) string {
	_ = title
	sheet := m.styles.Sheet
	if w := contentWidth(m.width); w > 0 {
		sheet = sheet.Width(w)
	}
	return sheet.Render(body)
}

func (m Model) footerStatusLine() string {
	if m.statusMsg != "" {
		return m.styles.Status.Render(m.statusMsg)
	}
	if m.view != viewHistory {
		return ""
	}
	historyFooter := "DoneDelayedRatio: " + fmtRatio(m.historyStats.DoneDelayedRatio) +
		"  AbandonedDelayedRatio: " + fmtRatio(m.historyStats.AbandonedDelayedRatio)
	return m.styles.Status.Render(historyFooter)
}

func fmtRatio(f float64) string {
	// Keep it simple for TUI.
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}

func (m Model) header(active string) string {
	left := m.styles.AppTitle.Render("[tick]")
	tabs := []string{
		m.tab("Today", active == "Today"),
		m.tab("Upcoming", active == "Upcoming"),
		m.tab("History", active == "History"),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  "+strings.Join(tabs, "  "))
}

func (m Model) tab(name string, on bool) string {
	if on {
		return m.styles.TabOn.Render("{" + name + "}")
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
