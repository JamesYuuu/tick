package domain

import (
	"fmt"
	"time"
)

type Day struct {
	t time.Time
}

const dayLayout = "2006-01-02"

func DayFromTime(t time.Time) Day {
	utc := t.UTC()
	return Day{t: time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)}
}

func ParseDay(s string) (Day, error) {
	t, err := time.ParseInLocation(dayLayout, s, time.UTC)
	if err != nil {
		return Day{}, fmt.Errorf("parse day: %w", err)
	}
	return DayFromTime(t), nil
}

func MustParseDay(s string) Day {
	d, err := ParseDay(s)
	if err != nil {
		panic(err)
	}
	return d
}

func (d Day) String() string {
	return d.t.Format(dayLayout)
}

func (d Day) Before(other Day) bool {
	return d.t.Before(other.t)
}
