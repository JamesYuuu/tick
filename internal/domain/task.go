package domain

type Status string

const (
	StatusActive Status = "active"
	StatusDone   Status = "done"
	StatusAbandoned Status = "abandoned"
)

type Task struct {
	ID           int64
	Title        string
	Status       Status
	CreatedDay   Day
	DueDay       Day // required by product; not intended to be zero-value
	DoneDay      *Day
	AbandonedDay *Day
}

func (t Task) IsDelayed(currentDay Day) bool {
	return t.Status == StatusActive && t.DueDay.Before(currentDay)
}
