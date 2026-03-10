package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

type styles struct {
	AppTitle lipgloss.Style
	Tab      lipgloss.Style
	Help     lipgloss.Style
	HelpKey  lipgloss.Style
	Stats    lipgloss.Style
	Sheet    lipgloss.Style
	Modal    lipgloss.Style
	RowSel   lipgloss.Style
	RowSelDl lipgloss.Style
	Reverse  lipgloss.Style
	Status   lipgloss.Style
	Delayed  lipgloss.Style
}

const maxContentWidth = 96
const appLogo = "tick"

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
		AppTitle: lipgloss.NewStyle().Bold(true).Italic(true),
		Tab:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Help:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		HelpKey:  lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true),
		Stats:    lipgloss.NewStyle().Foreground(lipgloss.Color("45")),
		Sheet:    lipgloss.NewStyle().Padding(0, 1).Border(sheetBorder).BorderForeground(lipgloss.Color("240")),
		Modal:    lipgloss.NewStyle().Padding(0, 1).Border(sheetBorder).BorderForeground(lipgloss.Color("240")),
		RowSel:   lipgloss.NewStyle().Background(lipgloss.Color("254")).Foreground(lipgloss.Color("236")),
		RowSelDl: lipgloss.NewStyle().Background(lipgloss.Color("254")).Foreground(lipgloss.Color("131")),
		Reverse:  lipgloss.NewStyle().Reverse(true),
		Status:   lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Delayed:  lipgloss.NewStyle().Foreground(lipgloss.Color("131")),
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
	return ""
}

func joinFooterParts(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		kept = append(kept, part)
	}
	return strings.Join(kept, "  ")
}

func truncateFooterPart(part string, width int) string {
	if width <= 0 || part == "" {
		return ""
	}
	if ansi.StringWidth(part) <= width {
		return part
	}
	return ansi.Truncate(part, width, "")
}

func formatTodayFooterLine1(windowWidth int, status, left, right string) string {
	width := contentWidth(windowWidth)
	if width <= 0 {
		return ""
	}

	leftPrefix := joinFooterParts(status, left)
	if leftPrefix == "" {
		return truncateFooterPart(right, width)
	}

	leftW := ansi.StringWidth(leftPrefix)
	rightW := ansi.StringWidth(right)
	if right != "" && leftW+2+rightW <= width {
		return leftPrefix + strings.Repeat(" ", width-leftW-rightW) + right
	}
	if leftW <= width {
		return leftPrefix
	}
	if status != "" {
		return truncateFooterPart(status, width)
	}
	return truncateFooterPart(left, width)
}

func (m Model) footerLines() (string, string) {
	helpLines := strings.Split(m.help(), "\n")
	line1 := ""
	line2 := ""
	if len(helpLines) > 0 {
		line1 = helpLines[0]
	}
	if len(helpLines) > 1 {
		line2 = helpLines[1]
	}
	status := m.footerStatusLine()
	if m.view == viewToday {
		left := m.helpLine(
			[2]string{"a", "add"},
			[2]string{"e", "edit"},
			[2]string{"d", "delete"},
		)
		right := m.helpLine(
			[2]string{"x", "done"},
			[2]string{"b", "abandon"},
			[2]string{"p", "postpone"},
		)
		return formatTodayFooterLine1(m.width, status, left, right), line2
	}
	if status == "" {
		return line1, line2
	}
	if line1 == "" {
		return status, line2
	}
	return status + "  " + line1, line2
}

func fmtRatio(f float64) string {
	// Keep it simple for TUI.
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", f), "0"), ".")
}

func (m Model) header(active string) string {
	left := m.styles.AppTitle.Render(appLogo)
	tabs := []string{
		m.tab("Today", active == "Today"),
		m.tab("Upcoming", active == "Upcoming"),
		m.tab("History", active == "History"),
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, "  "+strings.Join(tabs, "  "))
}

func (m Model) tab(name string, on bool) string {
	if on {
		return m.selectedLabel(name)
	}
	return m.styles.Tab.Render(name)
}

func (m Model) selectedLabel(label string) string {
	if lipgloss.ColorProfile() == termenv.Ascii {
		return "[" + label + "]"
	}
	return m.styles.Reverse.Render(label)
}

func (m Model) selectedCellLabel(label string) string {
	if lipgloss.ColorProfile() == termenv.Ascii {
		return m.selectedLabel(label)
	}
	return m.styles.Reverse.Render(" " + label + " ")
}

func (m Model) helpToken(key, label string) string {
	return m.styles.HelpKey.Render(key+":") + m.styles.Help.Render(label)
}

func (m Model) helpLine(tokens ...[2]string) string {
	parts := make([]string, 0, len(tokens))
	for _, token := range tokens {
		parts = append(parts, m.helpToken(token[0], token[1]))
	}
	return strings.Join(parts, "  ")
}

func alignHelpGroups(windowWidth int, left, right string) string {
	if left == "" {
		return right
	}
	if right == "" {
		return left
	}

	width := contentWidth(windowWidth)
	leftW := ansi.StringWidth(left)
	rightW := ansi.StringWidth(right)
	if width <= 0 || leftW+2+rightW > width {
		return left + "  " + right
	}

	pad := width - leftW - rightW
	if pad < 2 {
		pad = 2
	}
	return left + strings.Repeat(" ", pad) + right
}

func (m Model) help() string {
	line1 := ""
	line2 := m.helpLine([2]string{"tab", "next"}, [2]string{"q", "quit"})
	switch m.view {
	case viewToday:
		left := m.helpLine(
			[2]string{"a", "add"},
			[2]string{"e", "edit"},
			[2]string{"d", "delete"},
		)
		right := m.helpLine(
			[2]string{"x", "done"},
			[2]string{"b", "abandon"},
			[2]string{"p", "postpone"},
		)
		line1 = alignHelpGroups(m.width, left, right)
	case viewUpcoming:
		line1 = m.helpLine([2]string{"e", "edit"}, [2]string{"d", "delete"})
	case viewHistory:
		left := m.helpLine(
			[2]string{"left/h", "prev day"},
			[2]string{"right/l", "next day"},
		)
		right := m.helpLine(
			[2]string{"up/k", "scroll up"},
			[2]string{"down/j", "scroll down"},
		)
		line1 = alignHelpGroups(m.width, left, right)
	}
	return strings.Join([]string{line1, line2}, "\n")
}

func modalTextWidth(windowWidth int) int {
	w := contentWidth(windowWidth) - 24
	if w > 40 {
		return 40
	}
	if w < 12 {
		return 12
	}
	return w
}

func modalInputWidth(windowWidth int) int {
	w := modalTextWidth(windowWidth) - 2
	if w < 8 {
		return 8
	}
	return w
}

func renderOverlay(base, overlay string, width, height int) string {
	if overlay == "" {
		return base
	}

	canvasW := contentWidth(width)
	baseLines := strings.Split(forceHeight(base, height), "\n")
	overlayLines := strings.Split(strings.TrimRight(overlay, "\n"), "\n")
	if len(overlayLines) == 0 {
		return base
	}

	overlayW := 0
	for _, line := range overlayLines {
		if w := ansi.StringWidth(line); w > overlayW {
			overlayW = w
		}
	}
	leftPad := (canvasW - overlayW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (height - len(overlayLines)) / 2
	if topPad < 0 {
		topPad = 0
	}

	for i, line := range overlayLines {
		idx := topPad + i
		if idx < 0 || idx >= len(baseLines) {
			continue
		}
		baseLines[idx] = overlayCanvasLine(line, canvasW, leftPad)
	}
	return strings.Join(baseLines, "\n")
}

func overlayCanvasLine(overlay string, width, leftPad int) string {
	if width <= 0 {
		return overlay
	}
	if leftPad < 0 {
		leftPad = 0
	}
	available := width - leftPad
	if available < 0 {
		available = 0
	}
	line := ansi.Truncate(overlay, available, "")
	rightPad := width - leftPad - ansi.StringWidth(line)
	if rightPad < 0 {
		rightPad = 0
	}
	return strings.Repeat(" ", leftPad) + line + strings.Repeat(" ", rightPad)
}
