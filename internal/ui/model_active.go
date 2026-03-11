package ui

import (
	"context"
	"fmt"

	"github.com/JamesYuuu/tick/internal/domain"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) loadActiveLists(ctx context.Context, includeToday, includeUpcoming bool) (activeListsResult, error) {
	out := activeListsResult{hasToday: includeToday, hasUpcoming: includeUpcoming}
	if includeToday {
		today, err := m.app.Today(ctx)
		if err != nil {
			return activeListsResult{}, fmt.Errorf("today: %w", err)
		}
		out.today = today
	}
	if includeUpcoming {
		upcoming, err := m.app.Upcoming(ctx)
		if err != nil {
			return activeListsResult{}, fmt.Errorf("upcoming: %w", err)
		}
		out.upcoming = upcoming
	}
	return out, nil
}

func (m Model) cmdRefreshActive() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		lists, err := m.loadActiveLists(ctx, true, true)
		if err != nil {
			return refreshMsg{err: err}
		}
		return refreshMsg{today: lists.today, upcoming: lists.upcoming, hasToday: lists.hasToday, hasUpcoming: lists.hasUpcoming}
	}
}

func (m Model) cmdActThenRefresh(prefix string, act func(ctx context.Context) error) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := act(ctx); err != nil {
			return refreshMsg{err: fmt.Errorf("%s: %w", prefix, err)}
		}
		lists, err := m.loadActiveLists(ctx, true, true)
		if err != nil {
			return refreshMsg{err: err}
		}
		return refreshMsg{today: lists.today, upcoming: lists.upcoming, hasToday: lists.hasToday, hasUpcoming: lists.hasUpcoming}
	}
}

func (m *Model) applyActiveRefresh(todayTasks, upcomingTasks []domain.Task, hasToday, hasUpcoming bool) {
	m.statusMsg = ""
	if hasToday {
		m.todayList.SetItems(tasksToItems(todayTasks))
		today := m.currentDay()
		m.lastDay = today
		m.todayList.SetDelegate(todayItemDelegate{styles: m.styles, currentDay: today})
	}
	if hasUpcoming {
		m.upcomingList.SetItems(tasksToItems(upcomingTasks))
	}
}
