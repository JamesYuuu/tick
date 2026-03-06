package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"sync/atomic"

	_ "modernc.org/sqlite"

	"github.com/JamesYuuu/tick/internal/domain"
	"github.com/JamesYuuu/tick/internal/store"
)

var memDBCounter uint64

type SQLiteStore struct {
	db *sql.DB
}

const listActiveByCreatedDayQuery = "SELECT id, title, status, created_day, due_day, done_day, abandoned_day\n" +
	" FROM tasks\n" +
	" WHERE status = 'active' AND created_day = ?\n" +
	" ORDER BY id ASC"

func Open(path string) (*SQLiteStore, error) {
	dsn := dsnForPath(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := migrate(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func dsnForPath(path string) string {
	u := url.URL{Path: path}
	// Use URL path escaping so spaces are encoded, while keeping
	// the sqlite DSN format as "file:/abs/path".
	return "file:" + u.EscapedPath()
}

func OpenInMemory() (*SQLiteStore, error) {
	name := atomic.AddUint64(&memDBCounter, 1)
	dsn := fmt.Sprintf("file:memdb_%d?mode=memory&cache=shared", name)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite in memory: %w", err)
	}
	if err := migrate(context.Background(), db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func (s *SQLiteStore) CreateTask(ctx context.Context, title string, createdDay, dueDay domain.Day) (domain.Task, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO tasks(title, status, created_day, due_day, done_day, abandoned_day)
		 VALUES(?, ?, ?, ?, NULL, NULL)`,
		title, string(domain.StatusActive), createdDay.String(), dueDay.String(),
	)
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return domain.Task{}, fmt.Errorf("create task: %w", err)
	}
	return domain.Task{ID: id, Title: title, Status: domain.StatusActive, CreatedDay: createdDay, DueDay: dueDay}, nil
}

func (s *SQLiteStore) ListActive(ctx context.Context, p store.ListActiveParams) ([]domain.Task, error) {
	var op string
	switch p.Window {
	case store.ActiveDueLTECurrent:
		op = "<="
	case store.ActiveDueGTCurrent:
		op = ">"
	default:
		return nil, fmt.Errorf("list active: unknown window %d", p.Window)
	}

	q := fmt.Sprintf(`SELECT id, title, status, created_day, due_day, done_day, abandoned_day
		FROM tasks
		WHERE status = ? AND due_day %s ?
		ORDER BY due_day ASC, id ASC`, op)

	rows, err := s.db.QueryContext(ctx, q, string(domain.StatusActive), p.CurrentDay.String())
	if err != nil {
		return nil, fmt.Errorf("list active: %w", err)
	}
	defer rows.Close()

	var out []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list active: %w", err)
	}
	return out, nil
}

func (s *SQLiteStore) ListActiveByCreatedDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	return s.queryTasks(ctx, "list active by created day", listActiveByCreatedDayQuery, day.String())
}

func (s *SQLiteStore) MarkDone(ctx context.Context, id int64, doneDay domain.Day) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks
		 SET status = ?, done_day = ?, abandoned_day = NULL
		 WHERE id = ?`,
		string(domain.StatusDone), doneDay.String(), id,
	)
	if err != nil {
		return fmt.Errorf("mark done: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark done: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("mark done: id=%d: %w", id, sql.ErrNoRows)
	}
	return nil
}

func (s *SQLiteStore) MarkAbandoned(ctx context.Context, id int64, abandonedDay domain.Day) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks
		 SET status = ?, abandoned_day = ?, done_day = NULL
		 WHERE id = ?`,
		string(domain.StatusAbandoned), abandonedDay.String(), id,
	)
	if err != nil {
		return fmt.Errorf("mark abandoned: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("mark abandoned: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("mark abandoned: id=%d: %w", id, sql.ErrNoRows)
	}
	return nil
}

var ErrInvalidTransition = errors.New("invalid task status transition")

func (s *SQLiteStore) Postpone(ctx context.Context, id int64, newDueDay domain.Day) error {
	// Only allow postpone for active tasks.
	res, err := s.db.ExecContext(ctx,
		`UPDATE tasks
		 SET due_day = ?
		 WHERE id = ? AND status = ?`,
		newDueDay.String(), id, string(domain.StatusActive),
	)
	if err != nil {
		return fmt.Errorf("postpone: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postpone: %w", err)
	}
	if n == 0 {
		var exists int
		err := s.db.QueryRowContext(ctx, `SELECT 1 FROM tasks WHERE id = ?`, id).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("postpone: id=%d: %w", id, sql.ErrNoRows)
		}
		if err != nil {
			return fmt.Errorf("postpone: check exists: %w", err)
		}
		return ErrInvalidTransition
	}
	return nil
}

func (s *SQLiteStore) ListDoneByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	return s.queryTasks(ctx, "list done", `SELECT id, title, status, created_day, due_day, done_day, abandoned_day
		 FROM tasks
		 WHERE status = ? AND done_day = ?
		 ORDER BY id ASC`, string(domain.StatusDone), day.String())
}

func (s *SQLiteStore) ListAbandonedByDay(ctx context.Context, day domain.Day) ([]domain.Task, error) {
	return s.queryTasks(ctx, "list abandoned", `SELECT id, title, status, created_day, due_day, done_day, abandoned_day
		 FROM tasks
		 WHERE status = ? AND abandoned_day = ?
		 ORDER BY id ASC`, string(domain.StatusAbandoned), day.String())
}

func (s *SQLiteStore) queryTasks(ctx context.Context, op, q string, args ...any) ([]domain.Task, error) {
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var out []domain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	return out, nil
}

func (s *SQLiteStore) StatsOutcomeRatios(ctx context.Context, fromDay, toDay domain.Day) (store.OutcomeRatios, error) {
	var out store.OutcomeRatios

	if toDay.Before(fromDay) {
		return store.OutcomeRatios{}, fmt.Errorf("stats outcome ratios: invalid range: from=%s to=%s", fromDay.String(), toDay.String())
	}

	if err := s.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN due_day < done_day THEN 1 ELSE 0 END), 0) AS delayed
		 FROM tasks
		 WHERE status = ? AND done_day >= ? AND done_day <= ?`,
		string(domain.StatusDone), fromDay.String(), toDay.String(),
	).Scan(&out.TotalDone, &out.DelayedDone); err != nil {
		return store.OutcomeRatios{}, fmt.Errorf("stats outcome ratios: query done: %w", err)
	}

	if err := s.db.QueryRowContext(ctx,
		`SELECT
			COUNT(*) AS total,
			COALESCE(SUM(CASE WHEN due_day < abandoned_day THEN 1 ELSE 0 END), 0) AS delayed
		 FROM tasks
		 WHERE status = ? AND abandoned_day >= ? AND abandoned_day <= ?`,
		string(domain.StatusAbandoned), fromDay.String(), toDay.String(),
	).Scan(&out.TotalAbandoned, &out.DelayedAbandoned); err != nil {
		return store.OutcomeRatios{}, fmt.Errorf("stats outcome ratios: query abandoned: %w", err)
	}

	if out.TotalDone > 0 {
		out.DoneDelayedRatio = float64(out.DelayedDone) / float64(out.TotalDone)
	}
	if out.TotalAbandoned > 0 {
		out.AbandonedDelayedRatio = float64(out.DelayedAbandoned) / float64(out.TotalAbandoned)
	}

	return out, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanTask(r scannable) (domain.Task, error) {
	var (
		id            int64
		title         string
		status        string
		createdDayStr string
		dueDayStr     string
		doneDayStr    sql.NullString
		abDayStr      sql.NullString
	)
	if err := r.Scan(&id, &title, &status, &createdDayStr, &dueDayStr, &doneDayStr, &abDayStr); err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}
	createdDay, err := domain.ParseDay(createdDayStr)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}
	dueDay, err := domain.ParseDay(dueDayStr)
	if err != nil {
		return domain.Task{}, fmt.Errorf("scan task: %w", err)
	}
	var doneDay *domain.Day
	if doneDayStr.Valid {
		d, err := domain.ParseDay(doneDayStr.String)
		if err != nil {
			return domain.Task{}, fmt.Errorf("scan task: %w", err)
		}
		doneDay = &d
	}
	var abandonedDay *domain.Day
	if abDayStr.Valid {
		d, err := domain.ParseDay(abDayStr.String)
		if err != nil {
			return domain.Task{}, fmt.Errorf("scan task: %w", err)
		}
		abandonedDay = &d
	}

	return domain.Task{
		ID:           id,
		Title:        title,
		Status:       domain.Status(status),
		CreatedDay:   createdDay,
		DueDay:       dueDay,
		DoneDay:      doneDay,
		AbandonedDay: abandonedDay,
	}, nil
}
