from __future__ import annotations

from contextlib import closing
from dataclasses import dataclass
from datetime import date, datetime, timedelta
from pathlib import Path
import sqlite3


@dataclass(frozen=True)
class Task:
    id: int
    title: str
    status: str
    created_day: str
    due_day: str
    done_day: str | None
    abandoned_day: str | None


@dataclass(frozen=True)
class HistoryStats:
    total_done: int
    delayed_done: int
    total_abandoned: int
    delayed_abandoned: int
    done_delayed_ratio: float
    abandoned_delayed_ratio: float


@dataclass(frozen=True)
class HistorySnapshot:
    from_day: str
    to_day: str
    selected_day: str
    done: list[Task]
    abandoned: list[Task]
    active_created: list[Task]
    stats: HistoryStats


@dataclass(frozen=True)
class Snapshot:
    current_day: str
    today: list[Task]
    upcoming: list[Task]
    history: HistorySnapshot


class BackendError(RuntimeError):
    pass


class TickBackend:
    def __init__(self, db_path: Path | None = None) -> None:
        home = Path.home()
        self.data_dir = home / ".tick"
        self.db_path = db_path or (self.data_dir / "todo.db")
        self._ensure_database()

    def snapshot(self, history_day: str | None = None) -> Snapshot:
        current_day = self.current_day()
        selected_day = history_day or current_day
        from_day = (date.fromisoformat(selected_day) - timedelta(days=6)).isoformat()

        return Snapshot(
            current_day=current_day,
            today=self._list_active("<=", current_day),
            upcoming=self._list_active(">", current_day),
            history=HistorySnapshot(
                from_day=from_day,
                to_day=selected_day,
                selected_day=selected_day,
                done=self._list_done(selected_day),
                abandoned=self._list_abandoned(selected_day),
                active_created=self._list_active_created(selected_day),
                stats=self._stats(from_day, selected_day),
            ),
        )

    def add(self, title: str) -> Task:
        title = self._normalize_title(title)
        current_day = self.current_day()
        with closing(self._connect()) as conn:
            cursor = conn.execute(
                """
                INSERT INTO tasks(title, status, created_day, due_day, done_day, abandoned_day)
                VALUES (?, 'active', ?, ?, NULL, NULL)
                """,
                (title, current_day, current_day),
            )
            conn.commit()
            return Task(
                id=int(cursor.lastrowid),
                title=title,
                status="active",
                created_day=current_day,
                due_day=current_day,
                done_day=None,
                abandoned_day=None,
            )

    def edit(self, task_id: int, title: str) -> None:
        title = self._normalize_title(title)
        self._require_active_update(
            """
            UPDATE tasks
            SET title = ?
            WHERE id = ? AND status = 'active'
            """,
            (title, task_id),
            "edit task",
        )

    def delete(self, task_id: int) -> None:
        self._require_active_update(
            "DELETE FROM tasks WHERE id = ? AND status = 'active'",
            (task_id,),
            "delete task",
        )

    def done(self, task_id: int) -> None:
        current_day = self.current_day()
        self._require_active_update(
            """
            UPDATE tasks
            SET status = 'done', done_day = ?, abandoned_day = NULL
            WHERE id = ? AND status = 'active'
            """,
            (current_day, task_id),
            "mark done",
        )

    def abandon(self, task_id: int) -> None:
        current_day = self.current_day()
        self._require_active_update(
            """
            UPDATE tasks
            SET status = 'abandoned', abandoned_day = ?, done_day = NULL
            WHERE id = ? AND status = 'active'
            """,
            (current_day, task_id),
            "mark abandoned",
        )

    def postpone(self, task_id: int) -> None:
        current_day = date.fromisoformat(self.current_day())
        due_day = (current_day + timedelta(days=1)).isoformat()
        self._require_active_update(
            """
            UPDATE tasks
            SET due_day = ?
            WHERE id = ? AND status = 'active'
            """,
            (due_day, task_id),
            "postpone task",
        )

    def current_day(self) -> str:
        return datetime.now().astimezone().date().isoformat()

    def _list_active(self, operator: str, current_day: str) -> list[Task]:
        return self._query_tasks(
            f"""
            SELECT id, title, status, created_day, due_day, done_day, abandoned_day
            FROM tasks
            WHERE status = 'active' AND due_day {operator} ?
            ORDER BY due_day ASC, id ASC
            """,
            (current_day,),
        )

    def _list_active_created(self, selected_day: str) -> list[Task]:
        return self._query_tasks(
            """
            SELECT id, title, status, created_day, due_day, done_day, abandoned_day
            FROM tasks
            WHERE status = 'active' AND created_day = ?
            ORDER BY id ASC
            """,
            (selected_day,),
        )

    def _list_done(self, selected_day: str) -> list[Task]:
        return self._query_tasks(
            """
            SELECT id, title, status, created_day, due_day, done_day, abandoned_day
            FROM tasks
            WHERE status = 'done' AND done_day = ?
            ORDER BY id ASC
            """,
            (selected_day,),
        )

    def _list_abandoned(self, selected_day: str) -> list[Task]:
        return self._query_tasks(
            """
            SELECT id, title, status, created_day, due_day, done_day, abandoned_day
            FROM tasks
            WHERE status = 'abandoned' AND abandoned_day = ?
            ORDER BY id ASC
            """,
            (selected_day,),
        )

    def _stats(self, from_day: str, to_day: str) -> HistoryStats:
        with closing(self._connect()) as conn:
            done_total, done_delayed = conn.execute(
                """
                SELECT
                    COUNT(*) AS total,
                    COALESCE(SUM(CASE WHEN due_day < done_day THEN 1 ELSE 0 END), 0) AS delayed
                FROM tasks
                WHERE status = 'done' AND done_day >= ? AND done_day <= ?
                """,
                (from_day, to_day),
            ).fetchone()
            abandoned_total, abandoned_delayed = conn.execute(
                """
                SELECT
                    COUNT(*) AS total,
                    COALESCE(SUM(CASE WHEN due_day < abandoned_day THEN 1 ELSE 0 END), 0) AS delayed
                FROM tasks
                WHERE status = 'abandoned' AND abandoned_day >= ? AND abandoned_day <= ?
                """,
                (from_day, to_day),
            ).fetchone()

        return HistoryStats(
            total_done=int(done_total),
            delayed_done=int(done_delayed),
            total_abandoned=int(abandoned_total),
            delayed_abandoned=int(abandoned_delayed),
            done_delayed_ratio=(float(done_delayed) / float(done_total)) if done_total else 0.0,
            abandoned_delayed_ratio=(float(abandoned_delayed) / float(abandoned_total)) if abandoned_total else 0.0,
        )

    def _query_tasks(self, query: str, params: tuple[object, ...]) -> list[Task]:
        with closing(self._connect()) as conn:
            rows = conn.execute(query, params).fetchall()
        return [self._task_from_row(row) for row in rows]

    def _require_active_update(self, query: str, params: tuple[object, ...], action: str) -> None:
        with closing(self._connect()) as conn:
            cursor = conn.execute(query, params)
            conn.commit()
        if cursor.rowcount == 0:
            raise BackendError(f"{action}: task not found or not active")

    def _ensure_database(self) -> None:
        self.data_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
        self.db_path.touch(mode=0o600, exist_ok=True)
        with closing(self._connect()) as conn:
            conn.executescript(
                """
                CREATE TABLE IF NOT EXISTS tasks (
                    id INTEGER PRIMARY KEY,
                    title TEXT NOT NULL,
                    status TEXT NOT NULL CHECK (status IN ('active', 'done', 'abandoned')),
                    created_day TEXT NOT NULL,
                    due_day TEXT NOT NULL,
                    done_day TEXT NULL,
                    abandoned_day TEXT NULL
                );
                CREATE INDEX IF NOT EXISTS idx_tasks_status_due_day ON tasks(status, due_day);
                CREATE INDEX IF NOT EXISTS idx_tasks_done_day ON tasks(done_day);
                CREATE INDEX IF NOT EXISTS idx_tasks_abandoned_day ON tasks(abandoned_day);
                CREATE INDEX IF NOT EXISTS idx_tasks_status_done_day ON tasks(status, done_day);
                CREATE INDEX IF NOT EXISTS idx_tasks_status_abandoned_day ON tasks(status, abandoned_day);
                """
            )
            conn.commit()

    def _connect(self) -> sqlite3.Connection:
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        return conn

    @staticmethod
    def _normalize_title(title: str) -> str:
        normalized = title.strip()
        if not normalized:
            raise BackendError("title cannot be blank")
        return normalized

    @staticmethod
    def _task_from_row(row: sqlite3.Row) -> Task:
        return Task(
            id=int(row["id"]),
            title=str(row["title"]),
            status=str(row["status"]),
            created_day=str(row["created_day"]),
            due_day=str(row["due_day"]),
            done_day=row["done_day"],
            abandoned_day=row["abandoned_day"],
        )
