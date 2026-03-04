package domain

type Status string

const (
	StatusActive Status = "active"
	StatusDone   Status = "done"
)

type Task struct {
	Status Status
	DueDay Day
}

func (t Task) IsDelayed(currentDay Day) bool {
	return t.Status == StatusActive && t.DueDay.Before(currentDay)
}
