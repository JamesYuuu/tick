package ui

import (
	"fmt"
	"strings"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func renderTodayBody(m Model) string {
	if m.modal.kind == modalKindTask {
		g := calcLayoutMetrics(m.width, m.height)
		return centerInBox(renderModal(m), g.innerW, g.innerH)
	}
	if len(m.todayList.Items()) == 0 {
		return renderCenteredEmpty(m, "Nothing due today.")
	}
	return m.todayList.View()
}

func renderUpcomingBody(m Model) string {
	if m.modal.kind == modalKindTask {
		g := calcLayoutMetrics(m.width, m.height)
		return centerInBox(renderModal(m), g.innerW, g.innerH)
	}
	if len(m.upcomingList.Items()) == 0 {
		return renderCenteredEmpty(m, "No upcoming tasks.")
	}
	return m.upcomingList.View()
}

func renderCenteredEmpty(m Model, msg string) string {
	g := calcLayoutMetrics(m.width, m.height)
	return centerInBox(msg, g.innerW, g.innerH)
}

type historyLayout struct {
	innerW     int
	dateBlock  string
	divider    string
	statsBlock string
	detailH    int
}

func calcHistoryLayout(m Model) historyLayout {
	innerW := sheetInnerWidth(m.width)
	if innerW <= 0 {
		return historyLayout{}
	}

	dateBlock := renderHistoryDateTable(m, innerW)
	statsBlock := renderHistoryStats(m)
	divider := ""
	if innerW > 2 {
		divider = strings.Repeat("-", innerW-2)
	}

	innerH := calcLayoutMetrics(m.width, m.height).innerH
	selectorH := linesCount(dateBlock)
	detailH := innerH - (selectorH + 2 + historyStatsBlockHeight(m))
	if detailH < 0 {
		detailH = 0
	}

	return historyLayout{
		innerW:     innerW,
		dateBlock:  dateBlock,
		divider:    divider,
		statsBlock: statsBlock,
		detailH:    detailH,
	}
}

func renderHistoryBody(m Model) string {
	layout := calcHistoryLayout(m)
	if layout.innerW <= 0 {
		return ""
	}
	details := renderHistoryDetailsViewport(m, layout.detailH)

	parts := []string{layout.dateBlock, " ", layout.divider}
	if layout.detailH > 0 {
		parts = append(parts, details)
	}
	parts = append(parts, " ", layout.statsBlock)
	return strings.Join(parts, "\n")
}

func renderHistoryStats(m Model) string {
	line := fmt.Sprintf(
		"done: %d  abandoned: %d  overdue active: %d",
		len(m.historyDone),
		len(m.historyAbandoned),
		historyOverdueActiveCount(m),
	)
	return m.styles.Stats.Render(line)
}

func historyOverdueActiveCount(m Model) int {
	today := m.currentDay()
	count := 0
	for _, t := range m.historyActiveCreated {
		if t.Status != domain.StatusActive {
			continue
		}
		if !t.DueDay.Before(today) {
			continue
		}
		count++
	}
	return count
}

func (m Model) historyDetailViewportHeight() int {
	return calcHistoryLayout(m).detailH
}

func historyStatsBlockHeight(m Model) int {
	return linesCount(renderHistoryStats(m)) + 1
}

func (m Model) maxHistoryScroll() int {
	rows := historyDetailRows(m)
	h := m.historyDetailViewportHeight()
	if h <= 0 || len(rows) <= h {
		return 0
	}
	return len(rows) - h
}

func (m *Model) clampHistoryScroll() {
	if m.historyScroll < 0 {
		m.historyScroll = 0
	}
	max := m.maxHistoryScroll()
	if m.historyScroll > max {
		m.historyScroll = max
	}
}

func renderHistoryDateTable(m Model, innerW int) string {
	cellW := 7                // " 03-01 "
	tableW := 1 + 7*(cellW+1) // leading '+' + 7*(cell + '+')
	if tableW > innerW {
		// Fallback: 1-line date strip.
		parts := make([]string, 0, 7)
		for i := 0; i < 7; i++ {
			d := addDays(m.historyFrom, i)
			v := fmtMMDD(d)
			if i == m.historyIndex {
				v = m.selectedLabel(v)
			}
			parts = append(parts, v)
		}
		return centerLinesInWidth(strings.Join(parts, " "), innerW)
	}

	border := "+" + strings.Repeat("-", cellW)
	border = strings.Repeat(border, 7) + "+"

	row := make([]string, 0, 1+7)
	row = append(row, "|")
	for i := 0; i < 7; i++ {
		d := addDays(m.historyFrom, i)
		content := fmt.Sprintf(" %s ", fmtMMDD(d))
		if i == m.historyIndex {
			content = m.selectedCellLabel(fmtMMDD(d))
		}
		// Ensure fixed cell width (ANSI-aware width isn't needed since content is ASCII + SGR).
		if len(content) < cellW {
			content = content + strings.Repeat(" ", cellW-len(content))
		}
		row = append(row, content+"|")
	}
	line := strings.Join(row, "")

	block := strings.Join([]string{border, line, border}, "\n")
	return centerLinesInWidth(block, innerW)
}

func renderHistoryDetailsViewport(m Model, h int) string {
	rows := historyDetailRows(m)
	if h <= 0 {
		return ""
	}
	if len(rows) == 0 {
		rows = []string{"No history tasks."}
	}
	start := m.historyScroll
	if start < 0 {
		start = 0
	}
	maxStart := len(rows) - h
	if maxStart < 0 {
		maxStart = 0
	}
	if start > maxStart {
		start = maxStart
	}
	end := start + h
	if end > len(rows) {
		end = len(rows)
	}
	slice := rows[start:end]
	for len(slice) < h {
		slice = append(slice, " ")
	}
	return strings.Join(slice, "\n")
}

func historyDetailRows(m Model) []string {
	rows := make([]string, 0, len(m.historyDone)+len(m.historyAbandoned)+len(m.historyActiveCreated)+1)
	for _, t := range m.historyDone {
		line := "[✓] " + t.Title
		if t.DoneDay != nil && t.DueDay.Before(*t.DoneDay) {
			line = m.styles.Delayed.Render(line)
		}
		rows = append(rows, line)
	}
	for _, t := range m.historyAbandoned {
		line := "[✗] " + t.Title
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
		rows = append(rows, m.styles.Delayed.Render("[ ] "+t.Title))
	}
	return rows
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

func renderModal(m Model) string {
	switch m.modal.kind {
	case modalKindTask:
		return renderTaskModal(m)
	default:
		return ""
	}
}

func renderInputModal(m Model, title string) string {
	input := m.addInput
	input.Width = modalInputWidth(m.width)
	body := strings.Join([]string{
		title,
		"",
		input.View(),
		"",
		m.helpLine([2]string{"enter", "save"}, [2]string{"esc", "cancel"}),
	}, "\n")
	return m.styles.Modal.Render(body)
}

func renderTaskModal(m Model) string {
	contentWidth := taskModalContentWidth(m.width)
	dividerWidth := taskModalContentWidth(m.width) - 2

	bodyLines := []string{
		"",
		centerLinesInWidth(lipgloss.NewStyle().Bold(true).Render(taskModalTitle(m)), contentWidth),
		"",
		taskModalInputBlock(m),
		"",
		centerLinesInWidth(taskModalActionsLine(m), contentWidth),
		"",
	}
	if hint := taskModalHintLine(m, dividerWidth); hint != "" {
		bodyLines = append(bodyLines, centerLinesInWidth(hint, contentWidth))
	}
	bodyLines = append(bodyLines, taskModalDivider(dividerWidth), centerLinesInWidth(m.styles.Help.Render("tab:next action  enter:confirm  esc:close"), contentWidth))
	body := strings.Join(bodyLines, "\n")
	style := m.styles.Modal.Width(contentWidth).Height(len(bodyLines))
	return style.Render(body)
}

func taskModalTitle(m Model) string {
	if m.modal.taskID == 0 {
		return "Add Task"
	}
	return "Edit Task"
}

func taskModalInputView(m Model) string {
	input := m.addInput
	input.Width = taskModalInputWidth(m.width)
	bg := lipgloss.Color("239")
	input.PromptStyle = lipgloss.NewStyle().Background(bg)
	input.TextStyle = lipgloss.NewStyle().Background(bg)
	input.PlaceholderStyle = m.styles.Tab.Copy().Background(bg)
	input.Cursor.TextStyle = lipgloss.NewStyle().Background(bg)
	return input.View()
}

func taskModalInputBlock(m Model) string {
	blockWidth := taskModalInputBlockWidth(m.width)
	fill := lipgloss.NewStyle().
		Background(lipgloss.Color("239")).
		Width(blockWidth)
	fillInline := lipgloss.NewStyle().Background(lipgloss.Color("239"))
	line := taskModalInputView(m)
	line = strings.TrimRight(line, " ")
	lineWidth := ansi.StringWidth(line)
	if lineWidth > blockWidth {
		line = ansi.Truncate(line, blockWidth, "")
		lineWidth = blockWidth
	}
	padding := strings.Repeat(" ", blockWidth-lineWidth)

	return strings.Join([]string{
		fill.Render(""),
		line + fillInline.Render(padding),
		fill.Render(""),
	}, "\n")
}

func taskModalDivider(width int) string {
	if width <= 0 {
		return ""
	}
	return strings.Repeat("-", width)
}

func taskModalActionsLine(m Model) string {
	parts := []string{
		taskModalActionLabel(m, taskModalFocusSave, "Save", false),
		taskModalActionLabel(m, taskModalFocusCancel, "Cancel", false),
	}
	if m.modal.taskID != 0 {
		parts = append(parts, taskModalActionLabel(m, taskModalFocusDelete, "Delete", true))
	}
	return strings.Join(parts, "      ")
}

func taskModalActionLabel(m Model, focus taskModalFocus, label string, danger bool) string {
	token := "[" + label + "]"
	if danger {
		token = m.styles.Delayed.Render(token)
	}
	if m.modal.focus == focus {
		return m.styles.Reverse.Render(token)
	}
	return token
}

func taskModalHintLine(m Model, width int) string {
	if m.modal.focus != taskModalFocusDelete {
		return ""
	}
	return ansi.Truncate(m.styles.Delayed.Render("delete this task forever?"), width, "")
}

func renderDeleteModal(m Model) string {
	title := ansi.Truncate(m.modal.taskTitle, modalTextWidth(m.width), "")
	body := strings.Join([]string{
		"Delete task?",
		"",
		title,
		"",
		m.helpLine([2]string{"y", "delete"}, [2]string{"n", "cancel"}),
	}, "\n")
	return m.styles.Modal.Render(body)
}
