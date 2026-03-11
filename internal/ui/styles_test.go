package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestDefaultStyles_DelayedUsesTerracottaForeground(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	s := defaultStyles()

	if got := s.Delayed.Render("late"); !strings.Contains(got, "\x1b[38;5;131") {
		t.Fatalf("expected delayed style to use ANSI 256 terracotta foreground, got %q", got)
	}
	if got := s.RowSelDl.Render("late"); !strings.Contains(got, "\x1b[38;5;131") {
		t.Fatalf("expected selected delayed style to use ANSI 256 terracotta foreground, got %q", got)
	}
}

func TestTaskModalBodyWidth_ClampsToAvailableWidth(t *testing.T) {
	for _, windowWidth := range []int{20, 30, 40} {
		max := contentWidth(windowWidth) - 4
		if max < 0 {
			max = 0
		}
		if got := taskModalBodyWidth(windowWidth); got > max {
			t.Fatalf("expected task modal width <= %d at window width %d, got %d", max, windowWidth, got)
		}
	}
}
