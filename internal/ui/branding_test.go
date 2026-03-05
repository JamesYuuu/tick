package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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

func TestBranding_Frame_RendersSheetAndStableFooterWithStatus(t *testing.T) {
	lipgloss.SetColorProfile(termenv.Ascii)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	day := domain.DayFromTime(time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC))
	a := newFakeApp(day, nil)

	m := NewWithDeps(a, fakeClock{now: day.Time()}, time.UTC)
	m.statusMsg = "status: hello"

	out := m.View()

	if !strings.Contains(out, "q:Quit") {
		t.Fatalf("expected View to contain help line even with status set, got: %q", out)
	}
	if !strings.Contains(out, m.statusMsg) {
		t.Fatalf("expected View to contain status message, got: %q", out)
	}

	statusAt := strings.Index(out, m.statusMsg)
	helpAt := strings.Index(out, "q:Quit")
	if statusAt < 0 || helpAt < 0 || statusAt > helpAt {
		t.Fatalf("expected status to appear above help; statusAt=%d helpAt=%d output=%q", statusAt, helpAt, out)
	}

	// The refreshed layout wraps the body in an ASCII bordered "sheet".
	borderLine := regexp.MustCompile(`(?m)^\+-+\+$`)
	if !borderLine.MatchString(out) {
		t.Fatalf("expected View to include ASCII sheet border line like '+---+', got: %q", out)
	}
}
