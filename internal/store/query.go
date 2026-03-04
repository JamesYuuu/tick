package store

import "github.com/JamesYuuu/tick/internal/domain"

type ActiveWindow int

const (
	// ActiveDueLTECurrent selects active tasks due on or before the current day.
	ActiveDueLTECurrent ActiveWindow = iota
	// ActiveDueGTCurrent selects active tasks due after the current day.
	ActiveDueGTCurrent
)

type ListActiveParams struct {
	CurrentDay domain.Day
	Window     ActiveWindow
}
