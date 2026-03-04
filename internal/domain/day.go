package domain

import (
	"fmt"
	"time"
)

type Day struct {
	t time.Time
}

func ParseDay(s string) (Day, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Day{}, fmt.Errorf("parse day: %w", err)
	}
	return Day{t: t.UTC()}, nil
}

func MustParseDay(s string) Day {
	d, err := ParseDay(s)
	if err != nil {
		panic(err)
	}
	return d
}

func (d Day) String() string {
	return d.t.Format("2006-01-02")
}

func (d Day) Before(other Day) bool {
	return d.t.Before(other.t)
}
