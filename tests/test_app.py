from __future__ import annotations

import asyncio
import tempfile
import unittest
from pathlib import Path

from tick.app import TickTextualApp
from tick.backend import TickBackend


class FixedDayBackend(TickBackend):
    def __init__(self, db_path: Path, today: str) -> None:
        self._today = today
        super().__init__(db_path=db_path)

    def current_day(self) -> str:
        return self._today


class AppTests(unittest.IsolatedAsyncioTestCase):
    async def test_app_mounts_and_loads_snapshot(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            backend = FixedDayBackend(Path(temp_dir) / "todo.db", "2026-03-25")
            backend.add("Ship Python rewrite")

            app = TickTextualApp(backend)
            async with app.run_test() as pilot:
                await pilot.pause()
                self.assertIsNotNone(app.snapshot_data)
                self.assertEqual(len(app.today_rows), 1)
                self.assertEqual(app.today_rows[0].title, "Ship Python rewrite")
                self.assertIn("Ship Python rewrite", str(app.query_one("#today-focus").renderable))
                self.assertIn("Synced 2026-03-25", str(app.query_one("#flash").renderable))

    async def test_done_action_refreshes_today_list(self) -> None:
        with tempfile.TemporaryDirectory() as temp_dir:
            backend = FixedDayBackend(Path(temp_dir) / "todo.db", "2026-03-25")
            backend.add("Finish redesign")

            app = TickTextualApp(backend)
            async with app.run_test() as pilot:
                await pilot.pause()
                await pilot.press("x")
                await pilot.pause()
                self.assertEqual(len(app.today_rows), 0)
                self.assertIn("Synced 2026-03-25", str(app.query_one("#flash").renderable))
