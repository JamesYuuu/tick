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

	// domain.DayFromTime uses UTC calendar day. We need the calendar day in loc.
	now := c.Now().In(loc)
	return domain.MustParseDay(now.Format("2006-01-02"))
}
