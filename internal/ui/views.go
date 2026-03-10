package ui

import (
	"fmt"
	"strings"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/charmbracelet/x/ansi"
)

func renderTodayBody(m Model) string {
	if len(m.todayList.Items()) == 0 {
		return renderCenteredEmpty(m, "Nothing due today.")
	}
	return m.todayList.View()
}

func renderUpcomingBody(m Model) string {
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
	case modalKindAdd:
		return renderInputModal(m, "Add task")
	case modalKindEdit:
		return renderInputModal(m, "Edit task")
	case modalKindDelete:
		return renderDeleteModal(m)
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
