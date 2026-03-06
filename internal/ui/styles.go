package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type styles struct {
	AppTitle lipgloss.Style
	Tab      lipgloss.Style
	TabOn    lipgloss.Style
	Help     lipgloss.Style
	Sheet    lipgloss.Style
	RowSel   lipgloss.Style
	RowSelDl lipgloss.Style
	Reverse  lipgloss.Style
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
	cw := contentWidth(windowWidth)
	if cw <= 2 {
		return ""
	}
	// Global separators: keep left aligned, right -2 cells.
	return strings.Repeat("-", cw-2)
}

// (History-specific separators are not needed; fullscreen separators always
// align to the sheet frame, not to inner widgets.)

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

// centerInBox places the given multi-line block inside a w x h box by adding
// leading spaces (horizontal centering) and blank lines (vertical centering).
//
// It is ANSI-aware for width calculations.
func centerInBox(s string, w, h int) string {
	if w <= 0 || h <= 0 || s == "" {
		return s
	}

	trimmed := strings.TrimRight(s, "\n")
	lines := []string{""}
	if trimmed != "" {
		lines = strings.Split(trimmed, "\n")
	}

	contentW := 0
	for _, ln := range lines {
		if aw := ansi.StringWidth(ln); aw > contentW {
			contentW = aw
		}
	}
	contentH := len(lines)

	leftPad := (w - contentW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (h - contentH) / 2
	if topPad < 0 {
		topPad = 0
	}

	prefix := ""
	if leftPad > 0 {
		prefix = strings.Repeat(" ", leftPad)
	}

	out := make([]string, 0, h)
	for i := 0; i < topPad; i++ {
		out = append(out, " ")
	}
	for _, ln := range lines {
		out = append(out, prefix+ln)
	}
	if len(out) > h {
		out = out[:h]
	}
	for len(out) < h {
		out = append(out, " ")
	}
	return strings.Join(out, "\n")
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
		Reverse:  lipgloss.NewStyle().Reverse(true),
		Status:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Delayed:  lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	}
}

func centerLinesInWidth(block string, w int) string {
	if w <= 0 || block == "" {
		return block
	}
	lines := strings.Split(strings.TrimRight(block, "\n"), "\n")
	maxW := 0
	for _, ln := range lines {
		if aw := ansi.StringWidth(ln); aw > maxW {
			maxW = aw
		}
	}
	pad := (w - maxW) / 2
	if pad <= 0 {
		return strings.Join(lines, "\n")
	}
	prefix := strings.Repeat(" ", pad)
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
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
	base := "tab:Next  q:Quit"
	suffix := ""
	switch m.view {
	case viewToday:
		suffix = "  a:Add  x:Done  d:Abandon  p:+1 day"
	case viewHistory:
		suffix = "  left/h:prev day  right/l:next day  up/k:scroll up  down/j:scroll down"
	}
	return m.styles.Help.Render(base + suffix)
}
