from __future__ import annotations

from pathlib import Path

import pytest

from tick.app import TickTextualApp
from textual.widgets import DataTable

from conftest import FixedDayBackend, settle


async def open_history(pilot) -> None:
    await pilot.press("tab", "tab")
    await pilot.pause()


@pytest.mark.asyncio
async def test_history_navigation_stops_at_today(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)

        await pilot.press("left", "left")
        await settle(pilot)
        assert app.history_day == "2026-03-24"

        await pilot.press("right", "right", "right", "right")
        await settle(pilot, 3)

        assert app.history_day == "2026-03-26"


@pytest.mark.asyncio
async def test_history_navigation_reaches_earlier_dates(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)

        await pilot.press(*(["left"] * 10))
        await settle(pilot, 3)

        assert app.history_day == "2026-03-16"


@pytest.mark.asyncio
async def test_history_overdue_only_counts_carryovers_not_postponed_from_selected_day(
    tmp_path: Path,
) -> None:
    db_path = tmp_path / "todo.db"
    creator = FixedDayBackend(db_path, "2026-03-25")
    task = creator.add("Follow up vendor")
    creator.postpone(task.id)

    app = TickTextualApp(FixedDayBackend(db_path, "2026-03-27"))
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)
        await pilot.press("left", "left")
        await settle(pilot, 3)

        assert app.history_day == "2026-03-25"
        assert app.history_rows == []


@pytest.mark.asyncio
async def test_history_overdue_carryover_counts(tmp_path: Path) -> None:
    db_path = tmp_path / "todo.db"
    creator = FixedDayBackend(db_path, "2026-03-26")
    creator.add("Carry over")

    app = TickTextualApp(FixedDayBackend(db_path, "2026-03-27"))
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)
        await pilot.press("left")
        await settle(pilot, 3)

        assert app.history_day == "2026-03-26"
        assert len(app.history_rows) == 1
        assert app.history_rows[0].marker == "late"


@pytest.mark.asyncio
async def test_history_strip_scrolls_left(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)

        for _ in range(6):
            await pilot.press("left")
            await settle(pilot)

        assert app.history_day == "2026-03-20"


@pytest.mark.asyncio
async def test_history_navigation_clamps_at_min_date(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2020-01-04")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)

        for _ in range(10):
            await pilot.press("left")
            await settle(pilot)

        assert app.history_day is not None
        assert app.history_day >= backend.MIN_HISTORY_DATE.isoformat()


@pytest.mark.asyncio
async def test_history_columns_do_not_go_negative_in_narrow_layout(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")

    app = TickTextualApp(backend)
    async with app.run_test(size=(18, 18)) as pilot:
        await pilot.pause()
        await pilot.press("tab", "tab")
        await pilot.pause()

        table = app.query_one("#main-table", DataTable)
        assert table.columns["state"].width >= 0
        assert table.columns["task"].width >= 0
        assert table.columns["due"].width >= 0
