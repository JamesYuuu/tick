package timeutil

import "time"

// Clock abstracts time for deterministic tests.
type Clock interface {
	Now() time.Time
}

// SystemClock uses the real system time.
type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}
