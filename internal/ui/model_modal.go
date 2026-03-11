package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) cmdSubmitModal() tea.Cmd {
	modal := m.modal
	currentView := m.view
	title := strings.TrimSpace(m.addInput.Value())

	return func() tea.Msg {
		ctx := context.Background()
		if modal.taskID == 0 {
			if _, err := m.app.Add(ctx, title); err != nil {
				return modalSubmitMsg{err: fmt.Errorf("add: %w", err)}
			}
		} else {
			if err := m.app.EditTitle(ctx, modal.taskID, title); err != nil {
				return modalSubmitMsg{err: fmt.Errorf("edit: %w", err)}
			}
		}

		includeToday, includeUpcoming := false, false
		if modal.taskID == 0 {
			includeToday = true
		} else if currentView == viewUpcoming {
			includeUpcoming = true
		} else {
			includeToday = true
		}

		lists, err := m.loadActiveLists(ctx, includeToday, includeUpcoming)
		if err != nil {
			return modalSubmitMsg{err: err, close: true}
		}
		return modalSubmitMsg{
			today:       lists.today,
			upcoming:    lists.upcoming,
			hasToday:    lists.hasToday,
			hasUpcoming: lists.hasUpcoming,
			close:       true,
		}
	}
}

func (m Model) cmdConfirmDelete() tea.Cmd {
	modal := m.modal
	currentView := m.view

	return func() tea.Msg {
		ctx := context.Background()
		if err := m.app.Delete(ctx, modal.taskID); err != nil {
			return deleteModalSubmitMsg{view: currentView, err: fmt.Errorf("delete: %w", err)}
		}

		switch currentView {
		case viewToday:
			today, err := m.app.Today(ctx)
			if err != nil {
				return deleteModalSubmitMsg{view: currentView, err: fmt.Errorf("today: %w", err), close: true}
			}
			return deleteModalSubmitMsg{view: currentView, tasks: today, close: true}
		case viewUpcoming:
			upcoming, err := m.app.Upcoming(ctx)
			if err != nil {
				return deleteModalSubmitMsg{view: currentView, err: fmt.Errorf("upcoming: %w", err), close: true}
			}
			return deleteModalSubmitMsg{view: currentView, tasks: upcoming, close: true}
		default:
			return deleteModalSubmitMsg{view: currentView, close: true}
		}
	}
}

func (m *Model) openAddModal() {
	m.modal = modalState{kind: modalKindTask, focus: taskModalFocusTitle}
	m.addInput.SetValue("")
	m.addInput.Width = taskModalInputWidth(m.width)
	m.setTaskModalFocus(taskModalFocusTitle)
}

func (m *Model) openEditTaskModal(task domain.Task) {
	m.modal = modalState{
		kind:   modalKindTask,
		taskID: task.ID,
		focus:  taskModalFocusTitle,
	}
	m.addInput.SetValue(task.Title)
	m.addInput.Width = taskModalInputWidth(m.width)
	m.setTaskModalFocus(taskModalFocusTitle)
}

func (m *Model) openDeleteTaskModal(task domain.Task) {
	m.modal = modalState{
		kind:   modalKindTask,
		taskID: task.ID,
		focus:  taskModalFocusDelete,
	}
	m.addInput.SetValue(task.Title)
	m.addInput.Width = taskModalInputWidth(m.width)
	m.setTaskModalFocus(taskModalFocusDelete)
}

func (m *Model) closeModal() {
	m.modal = modalState{}
	m.addInput.Blur()
	m.addInput.SetValue("")
	m.addInput.Width = m.modalInputWidth()
}

func (m *Model) setTaskModalFocus(focus taskModalFocus) {
	m.modal.focus = focus
	if focus == taskModalFocusTitle {
		m.addInput.Focus()
		return
	}
	m.addInput.Blur()
}

func (m *Model) advanceTaskModalFocus() {
	next := m.modal.focus + 1
	if next > taskModalFocusDelete {
		next = taskModalFocusTitle
	}
	if m.modal.taskID == 0 && next == taskModalFocusDelete {
		next = taskModalFocusTitle
	}
	m.setTaskModalFocus(next)
}

func (m Model) modalInputWidth() int {
	if m.modal.kind == modalKindTask {
		return taskModalInputWidth(m.width)
	}
	return sheetInnerWidth(m.width)
}

func modalBlocksKey(keys keyMap, msg tea.KeyMsg) bool {
	return key.Matches(msg, keys.Done) ||
		key.Matches(msg, keys.Abandon) ||
		key.Matches(msg, keys.Postpone) ||
		key.Matches(msg, keys.Edit) ||
		key.Matches(msg, keys.Delete) ||
		key.Matches(msg, keys.Add) ||
		key.Matches(msg, keys.NextView)
}

func (m Model) handleModalKey(msg tea.KeyMsg) (Model, tea.Cmd, bool) {
	if m.modal.kind == modalKindNone {
		return m, nil, false
	}

	if msg.Type == tea.KeyEsc {
		m.closeModal()
		return m, nil, true
	}

	if msg.Type == tea.KeyTab {
		m.advanceTaskModalFocus()
		m.addInput.Width = taskModalInputWidth(m.width)
		if m.modal.focus == taskModalFocusTitle {
			return m, m.addInput.Focus(), true
		}
		return m, nil, true
	}
	if msg.Type == tea.KeyEnter {
		switch m.modal.focus {
		case taskModalFocusTitle:
			return m, nil, true
		case taskModalFocusSave:
			if m.modal.submitting || strings.TrimSpace(m.addInput.Value()) == "" {
				return m, nil, true
			}
			m.modal.submitting = true
			return m, m.cmdSubmitModal(), true
		case taskModalFocusCancel:
			m.closeModal()
			return m, nil, true
		case taskModalFocusDelete:
			if m.modal.submitting {
				return m, nil, true
			}
			m.modal.submitting = true
			return m, m.cmdConfirmDelete(), true
		}
	}
	if m.modal.focus == taskModalFocusTitle {
		var cmd tea.Cmd
		m.addInput, cmd = m.addInput.Update(msg)
		return m, cmd, true
	}
	if modalBlocksKey(m.keys, msg) {
		return m, nil, true
	}
	return m, nil, true
}
