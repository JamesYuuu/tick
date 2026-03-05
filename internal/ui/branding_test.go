package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
)

func TestBranding_HeaderShowsTick(t *testing.T) {
	day := domain.DayFromTime(time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC))
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: day.Time()}, time.UTC)

	out := m.View()
	if !strings.Contains(out, "tick") {
		t.Fatalf("expected View to contain app name 'tick', got: %q", out)
	}
	if strings.Contains(out, "tuitodo") {
		t.Fatalf("expected View to not contain old name 'tuitodo', got: %q", out)
	}
}
