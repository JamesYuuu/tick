from __future__ import annotations
from contextlib import closing
from dataclasses import dataclass
from datetime import date, datetime, timedelta
from pathlib import Path
import os
import sqlite3
from typing import Literal

_TASK_COLUMNS = "id, title, status, created_day, due_day, done_day, abandoned_day"


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
    active_due: list[Task]
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
    MIN_HISTORY_DATE: date = date(2020, 1, 1)

    def __init__(self, db_path: Path | None = None) -> None:
        resolved = Path(db_path).expanduser() if db_path is not None else self.default_db_path()
        self.data_dir = resolved.parent
        self.db_path = resolved
        self._ensure_database()

    @classmethod
    def default_db_path(cls) -> Path:
        data_home = os.environ.get("XDG_DATA_HOME")
        if data_home:
            return Path(data_home).expanduser() / "tick" / "todo.db"
        return Path.home() / ".tick" / "todo.db"

    def snapshot(self, history_day: str | None = None) -> Snapshot:
        current_day = self.current_day()
        selected_day = (
            self._normalize_day(history_day, field_name="history_day")
            if history_day
            else current_day
        )
        if date.fromisoformat(selected_day) > date.fromisoformat(current_day):
            raise BackendError("history_day cannot be in the future")
        from_day = (date.fromisoformat(selected_day) - timedelta(days=6)).isoformat()
        with closing(self._connect()) as conn:
            conn.execute("BEGIN")
            return Snapshot(
                current_day=current_day,
                today=self._list_active_conn(conn, "<=", current_day),
                upcoming=self._list_active_conn(conn, ">", current_day),
                history=HistorySnapshot(
                    from_day=from_day,
                    to_day=selected_day,
                    selected_day=selected_day,
                    done=self._list_done_conn(conn, selected_day),
                    abandoned=self._list_abandoned_conn(conn, selected_day),
                    active_due=self._list_active_due_conn(conn, selected_day, current_day),
                    stats=self._stats_conn(conn, from_day, selected_day),
                ),
            )

    def add(self, title: str) -> Task:
        title = self._normalize_title(title)
        current_day = self.current_day()
        with closing(self._connect()) as conn:
            cursor = conn.execute(
                "INSERT INTO tasks(title, status, created_day, due_day, done_day, abandoned_day)"
                " VALUES (?, 'active', ?, ?, NULL, NULL)",
                (title, current_day, current_day),
            )
            conn.commit()
            return Task(
                id=int(cursor.lastrowid or 0),
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
            "UPDATE tasks SET title = ? WHERE id = ? AND status = 'active'",
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
            "UPDATE tasks SET status = 'done', done_day = ?, abandoned_day = NULL"
            " WHERE id = ? AND status = 'active'",
            (current_day, task_id),
            "mark done",
        )

    def abandon(self, task_id: int) -> None:
        current_day = self.current_day()
        self._require_active_update(
            "UPDATE tasks SET status = 'abandoned', abandoned_day = ?, done_day = NULL"
            " WHERE id = ? AND status = 'active'",
            (current_day, task_id),
            "mark abandoned",
        )

    def postpone(self, task_id: int) -> None:
        current_day = date.fromisoformat(self.current_day())
        with closing(self._connect()) as conn:
            row = conn.execute(
                f"SELECT {_TASK_COLUMNS} FROM tasks WHERE id = ?", (task_id,)
            ).fetchone()
            if row is None or row["status"] != "active":
                raise BackendError("postpone task: task not found or not active")
            base_day = max(current_day, date.fromisoformat(str(row["due_day"])))
            due_day = (base_day + timedelta(days=1)).isoformat()
            cursor = conn.execute(
                "UPDATE tasks SET due_day = ? WHERE id = ? AND status = 'active'",
                (due_day, task_id),
            )
            conn.commit()
            if cursor.rowcount == 0:
                raise BackendError("postpone task: task not found or not active")

    def current_day(self) -> str:
        return datetime.now().astimezone().date().isoformat()

    def _list_active_conn(
        self, conn: sqlite3.Connection, op: Literal["<=", ">"], current_day: str
    ) -> list[Task]:
        return self._query_by_status_conn(
            conn,
            f"status = 'active' AND due_day {op} ?",
            (current_day,),
            order_by="due_day ASC, id ASC",
        )

    def _list_done_conn(self, conn: sqlite3.Connection, selected_day: str) -> list[Task]:
        return self._query_by_status_conn(conn, "status = 'done' AND done_day = ?", (selected_day,))

    def _list_abandoned_conn(self, conn: sqlite3.Connection, selected_day: str) -> list[Task]:
        return self._query_by_status_conn(
            conn, "status = 'abandoned' AND abandoned_day = ?", (selected_day,)
        )

    def _list_active_due_conn(
        self, conn: sqlite3.Connection, selected_day: str, current_day: str
    ) -> list[Task]:
        return self._query_by_status_conn(
            conn,
            "status = 'active' AND due_day = ? AND due_day < ?",
            (selected_day, current_day),
        )

    def _query_by_status_conn(
        self,
        conn: sqlite3.Connection,
        where: str,
        params: tuple[object, ...],
        *,
        order_by: str = "id ASC",
    ) -> list[Task]:
        query = f"SELECT {_TASK_COLUMNS} FROM tasks WHERE {where} ORDER BY {order_by}"
        rows = conn.execute(query, params).fetchall()
        return [self._task_from_row(row) for row in rows]

    def _stats_conn(self, conn: sqlite3.Connection, from_day: str, to_day: str) -> HistoryStats:
        done_total, done_delayed = conn.execute(
            "SELECT COUNT(*) AS total,"
            " COALESCE(SUM(CASE WHEN due_day < done_day THEN 1 ELSE 0 END), 0) AS delayed"
            " FROM tasks WHERE status = 'done' AND done_day >= ? AND done_day <= ?",
            (from_day, to_day),
        ).fetchone()
        abandoned_total, abandoned_delayed = conn.execute(
            "SELECT COUNT(*) AS total,"
            " COALESCE(SUM(CASE WHEN due_day < abandoned_day THEN 1 ELSE 0 END), 0) AS delayed"
            " FROM tasks WHERE status = 'abandoned' AND abandoned_day >= ? AND abandoned_day <= ?",
            (from_day, to_day),
        ).fetchone()
        return HistoryStats(
            total_done=int(done_total),
            delayed_done=int(done_delayed),
            total_abandoned=int(abandoned_total),
            delayed_abandoned=int(abandoned_delayed),
            done_delayed_ratio=(float(done_delayed) / float(done_total)) if done_total else 0.0,
            abandoned_delayed_ratio=(float(abandoned_delayed) / float(abandoned_total))
            if abandoned_total
            else 0.0,
        )

    def _require_active_update(self, query: str, params: tuple[object, ...], action: str) -> None:
        with closing(self._connect()) as conn:
            cursor = conn.execute(query, params)
            affected = cursor.rowcount
            conn.commit()
        if affected == 0:
            raise BackendError(f"{action}: task not found or not active")

    def _ensure_database(self) -> None:
        self.data_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
        self.db_path.touch(mode=0o600, exist_ok=True)
        with closing(self._connect()) as conn:
            conn.executescript("""
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
            """)
            conn.commit()

    def _connect(self) -> sqlite3.Connection:
        conn = sqlite3.connect(self.db_path)
        conn.row_factory = sqlite3.Row
        return conn

    @staticmethod
    def _normalize_title(title: str) -> str:
        normalized = " ".join(title.split())
        if not normalized:
            raise BackendError("title cannot be blank")
        return normalized

    @staticmethod
    def _normalize_day(value: str, *, field_name: str) -> str:
        try:
            return date.fromisoformat(value).isoformat()
        except ValueError as exc:
            raise BackendError(f"{field_name} must be a valid ISO date") from exc

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
