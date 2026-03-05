package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"

	"github.com/JamesYuuu/tick/internal/app"
	"github.com/JamesYuuu/tick/internal/store/sqlite"
	"github.com/JamesYuuu/tick/internal/timeutil"
	"github.com/JamesYuuu/tick/internal/ui"
)

func main() {
	if !isatty.IsTerminal(os.Stdin.Fd()) || !isatty.IsTerminal(os.Stdout.Fd()) {
		fmt.Fprintln(os.Stderr, "tick: must be run in a TTY")
		os.Exit(2)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "tick: resolve home dir:", err)
		os.Exit(1)
	}
	dataDir := filepath.Join(home, ".tick")
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		fmt.Fprintln(os.Stderr, "tick: create data dir:", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(dataDir, "todo.db")

	f, err := os.OpenFile(dbPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tick: create db file:", err)
		os.Exit(1)
	}
	_ = f.Close()

	s, err := sqlite.Open(dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tick: open sqlite:", err)
		os.Exit(1)
	}
	defer func() { _ = s.Close() }()

	a, err := app.New(app.Config{Store: s, Clock: timeutil.SystemClock{}, Location: time.Local})
	if err != nil {
		fmt.Fprintln(os.Stderr, "tick: init app:", err)
		os.Exit(1)
	}

	p := tea.NewProgram(ui.New(a), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tick: run:", err)
		os.Exit(1)
	}
}
