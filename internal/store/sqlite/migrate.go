package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

func migrate(ctx context.Context, db *sql.DB) error {
	const schema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
	version INTEGER NOT NULL PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS tasks (
	id INTEGER PRIMARY KEY,
	title TEXT NOT NULL,
	status TEXT NOT NULL,
	created_day TEXT NOT NULL,
	due_day TEXT NOT NULL,
	done_day TEXT NULL,
	abandoned_day TEXT NULL
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_due_day ON tasks(status, due_day);
CREATE INDEX IF NOT EXISTS idx_tasks_done_day ON tasks(done_day);
CREATE INDEX IF NOT EXISTS idx_tasks_abandoned_day ON tasks(abandoned_day);
`

	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
