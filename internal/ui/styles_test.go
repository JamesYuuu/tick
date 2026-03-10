package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
)

func TestRenderOverlay_ReplacesCenteredLinesSafelyWithANSI(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	t.Cleanup(func() { lipgloss.SetColorProfile(termenv.Ascii) })

	baseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	overlayStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)

	baseLines := []string{
		baseStyle.Render("alpha row"),
		baseStyle.Render("bravo row"),
		baseStyle.Render("charlie row"),
		baseStyle.Render("delta row"),
		baseStyle.Render("echo row"),
	}
	overlayLines := []string{
		overlayStyle.Render("MODAL"),
		overlayStyle.Render("LINES"),
	}

	got := renderOverlay(strings.Join(baseLines, "\n"), strings.Join(overlayLines, "\n"), 20, len(baseLines))
	lines := strings.Split(got, "\n")

	if len(lines) != len(baseLines) {
		t.Fatalf("expected %d lines, got %d", len(baseLines), len(lines))
	}

	topPad := (len(baseLines) - len(overlayLines)) / 2
	leftPad := (contentWidth(20) - ansi.StringWidth(overlayLines[0])) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	prefix := strings.Repeat(" ", leftPad)
	suffix := strings.Repeat(" ", contentWidth(20)-leftPad-ansi.StringWidth(overlayLines[0]))

	if lines[topPad] != prefix+overlayLines[0]+suffix {
		t.Fatalf("expected centered overlay line %q, got %q", prefix+overlayLines[0]+suffix, lines[topPad])
	}
	if lines[topPad+1] != prefix+overlayLines[1]+suffix {
		t.Fatalf("expected centered overlay line %q, got %q", prefix+overlayLines[1]+suffix, lines[topPad+1])
	}

	if lines[0] != baseLines[0] {
		t.Fatalf("expected non-overlay line to stay unchanged, got %q", lines[0])
	}
	if lines[len(lines)-1] != baseLines[len(baseLines)-1] {
		t.Fatalf("expected non-overlay line to stay unchanged, got %q", lines[len(lines)-1])
	}
}

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
