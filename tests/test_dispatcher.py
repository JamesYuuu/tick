from __future__ import annotations

import threading
import time
from pathlib import Path

import pytest

from tick.app import TickTextualApp

from conftest import FixedDayBackend


class SlowSnapshotBackend(FixedDayBackend):
    def __init__(self, db_path: Path, today: str, delay: float = 0.05) -> None:
        self.delay = delay
        self.active_snapshots = 0
        self.max_active_snapshots = 0
        self._lock = threading.Lock()
        super().__init__(db_path, today)

    def snapshot(self, history_day: str | None = None):
        with self._lock:
            self.active_snapshots += 1
            self.max_active_snapshots = max(self.max_active_snapshots, self.active_snapshots)
        try:
            time.sleep(self.delay)
            return super().snapshot(history_day)
        finally:
            with self._lock:
                self.active_snapshots -= 1


@pytest.mark.asyncio
async def test_repeated_refreshes_do_not_run_snapshots_concurrently(tmp_path: Path) -> None:
    backend = SlowSnapshotBackend(tmp_path / "todo.db", "2026-03-25")
    backend.add("Ship Python rewrite")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        for _ in range(5):
            await pilot.pause()
        await pilot.press("r", "r", "r", "r")
        for _ in range(20):
            await pilot.pause()

        assert backend.max_active_snapshots == 1
