package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

func migrate(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("migrate: begin: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY,
			title TEXT NOT NULL,
			status TEXT NOT NULL CHECK (status IN ('active', 'done', 'abandoned')),
			created_day TEXT NOT NULL,
			due_day TEXT NOT NULL,
			done_day TEXT NULL,
			abandoned_day TEXT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status_due_day ON tasks(status, due_day);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_done_day ON tasks(done_day);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_abandoned_day ON tasks(abandoned_day);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status_done_day ON tasks(status, done_day);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status_abandoned_day ON tasks(status, abandoned_day);`,
	}

	for _, stmt := range stmts {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate: commit: %w", err)
	}
	return nil
}
