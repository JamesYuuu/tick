from __future__ import annotations

import tempfile
import unittest
from pathlib import Path

from tick.backend import BackendError, TickBackend


class FixedDayBackend(TickBackend):
    def __init__(self, db_path: Path, today: str) -> None:
        self._today = today
        super().__init__(db_path=db_path)

    def current_day(self) -> str:
        return self._today


class BackendTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp_dir = tempfile.TemporaryDirectory()
        self.db_path = Path(self.temp_dir.name) / "todo.db"
        self.backend = FixedDayBackend(self.db_path, "2026-03-25")

    def tearDown(self) -> None:
        self.temp_dir.cleanup()

    def test_add_and_snapshot_places_task_in_today(self) -> None:
        task = self.backend.add("Write migration plan")
        snapshot = self.backend.snapshot()

        self.assertEqual(task.title, "Write migration plan")
        self.assertEqual(len(snapshot.today), 1)
        self.assertEqual(snapshot.today[0].title, "Write migration plan")
        self.assertEqual(snapshot.upcoming, [])

    def test_postpone_moves_task_to_upcoming(self) -> None:
        task = self.backend.add("Review schema")
        self.backend.postpone(task.id)

        snapshot = self.backend.snapshot()

        self.assertEqual(snapshot.today, [])
        self.assertEqual(len(snapshot.upcoming), 1)
        self.assertEqual(snapshot.upcoming[0].due_day, "2026-03-26")

    def test_done_updates_history_and_stats(self) -> None:
        task = self.backend.add("Close migration")
        self.backend.done(task.id)

        snapshot = self.backend.snapshot("2026-03-25")

        self.assertEqual(len(snapshot.history.done), 1)
        self.assertEqual(snapshot.history.done[0].title, "Close migration")
        self.assertEqual(snapshot.history.stats.total_done, 1)
        self.assertEqual(snapshot.history.stats.delayed_done, 0)

    def test_blank_title_is_rejected(self) -> None:
        with self.assertRaises(BackendError):
            self.backend.add("   ")
