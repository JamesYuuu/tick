package ui

import (
	"context"

	"github.com/JamesYuuu/tick/internal/app"
	"github.com/JamesYuuu/tick/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
)

type historyRefreshMsg struct {
	done          []domain.Task
	abandoned     []domain.Task
	activeCreated []domain.Task
	stats         app.OutcomeRatios
	hasStats      bool
	err           error
}

func (m Model) historySelectedDay() domain.Day {
	return addDays(m.historyFrom, m.historyIndex)
}

func (m Model) cmdRefreshHistory(withStats bool) tea.Cmd {
	day := m.historySelectedDay()
	from, to := m.historyFrom, m.historyTo
	return func() tea.Msg {
		ctx := context.Background()
		done, err := m.app.HistoryDoneByDay(ctx, day)
		if err != nil {
			return historyRefreshMsg{err: err}
		}
		abandoned, err := m.app.HistoryAbandonedByDay(ctx, day)
		if err != nil {
			return historyRefreshMsg{err: err}
		}
		activeCreated, err := m.app.HistoryActiveByCreatedDay(ctx, day)
		if err != nil {
			return historyRefreshMsg{err: err}
		}
		if !withStats {
			return historyRefreshMsg{done: done, abandoned: abandoned, activeCreated: activeCreated}
		}
		stats, err := m.app.Stats(ctx, from, to)
		if err != nil {
			return historyRefreshMsg{err: err}
		}
		return historyRefreshMsg{done: done, abandoned: abandoned, activeCreated: activeCreated, stats: stats, hasStats: true}
	}
}

func (m Model) cmdRefreshHistorySelectedDay() tea.Cmd {
	return m.cmdRefreshHistory(false)
}

func (m Model) cmdRefreshHistoryWithStats() tea.Cmd {
	return m.cmdRefreshHistory(true)
}
