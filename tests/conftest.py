from __future__ import annotations

from pathlib import Path

from tick.backend import TickBackend


class FixedDayBackend(TickBackend):
    """Test backend that returns a fixed ``today`` date instead of the real clock."""

    def __init__(self, db_path: Path, today: str) -> None:
        self._today = today
        super().__init__(db_path=db_path)

    def current_day(self) -> str:
        return self._today


async def settle(pilot, pauses: int = 1) -> None:
    for _ in range(pauses):
        await pilot.pause()
