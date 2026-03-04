package timeutil

import (
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
)

// CurrentDay returns the current calendar day in the provided location,
// normalized to midnight UTC via domain.Day.
func CurrentDay(c Clock, loc *time.Location) domain.Day {
	if loc == nil {
		loc = time.UTC
	}

	now := c.Now().In(loc)
	localMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	return domain.DayFromTime(localMidnight)
}
