package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/JamesYuuu/tick/internal/app"
	"github.com/JamesYuuu/tick/internal/store/sqlite"
	"github.com/JamesYuuu/tick/internal/timeutil"
	"github.com/JamesYuuu/tick/internal/ui"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tuitodo: resolve home dir:", err)
		os.Exit(1)
	}
	dataDir := filepath.Join(home, ".tuitodo")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "tuitodo: create data dir:", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(dataDir, "todo.db")

	s, err := sqlite.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tuitodo: open sqlite:", err)
		os.Exit(1)
	}
	defer func() { _ = s.Close() }()

	a, err := app.New(app.Config{Store: s, Clock: timeutil.SystemClock{}, Location: time.Local})
	if err != nil {
		fmt.Fprintln(os.Stderr, "tuitodo: init app:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(a), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tuitodo: run:", err)
		os.Exit(1)
	}
}
