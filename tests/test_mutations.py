from __future__ import annotations

from pathlib import Path

import pytest

from tick.app import TickTextualApp
from textual.widgets import DataTable, Input

from conftest import FixedDayBackend, settle


@pytest.fixture()
def backend(tmp_path: Path) -> FixedDayBackend:
    return FixedDayBackend(tmp_path / "todo.db", "2026-03-25")


@pytest.mark.asyncio
async def test_done_removes_task_from_today(backend: FixedDayBackend) -> None:
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        await pilot.press("x")
        for _ in range(3):
            await pilot.pause()
        assert len(app.today_rows) == 0


@pytest.mark.asyncio
async def test_abandon_moves_task_into_history(backend: FixedDayBackend) -> None:
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        await pilot.press("b")
        for _ in range(3):
            await pilot.pause()

        assert len(app.today_rows) == 0

        await pilot.press("tab", "tab")
        await pilot.pause()

        assert len(app.history_rows) == 1
        assert app.history_rows[0].marker == "drop"
        assert app.history_rows[0].title == "Finish redesign"


@pytest.mark.asyncio
async def test_add_creates_task(backend: FixedDayBackend) -> None:
    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()

        await pilot.press("a")
        await pilot.pause()
        task_input = app.screen.query_one("#task-input", Input)
        task_input.value = "Write regression test"

        await pilot.press("enter")
        for _ in range(4):
            await pilot.pause()

        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Write regression test"


@pytest.mark.asyncio
async def test_cancel_paths_do_not_mutate_state(backend: FixedDayBackend) -> None:
    backend.add("Draft summary")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()

        await pilot.press("a")
        await pilot.pause()
        await pilot.press("escape")
        await pilot.pause()
        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Draft summary"

        await pilot.press("e")
        await pilot.pause()
        await pilot.press("escape")
        await pilot.pause()
        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Draft summary"

        await pilot.press("d")
        await pilot.pause()
        await pilot.press("n")
        await pilot.pause()
        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Draft summary"


@pytest.mark.asyncio
async def test_edit_then_postpone_then_delete_lifecycle(
    backend: FixedDayBackend,
) -> None:
    backend.add("Draft sprint summary")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()

        await pilot.press("e")
        await pilot.pause()
        task_input = app.screen.query_one("#task-input", Input)
        task_input.value = "  Polish   sprint   summary  "
        await pilot.press("enter")
        for _ in range(3):
            await pilot.pause()
        assert app.today_rows[0].title == "Polish sprint summary"

        await pilot.press("p")
        for _ in range(3):
            await pilot.pause()
        assert len(app.today_rows) == 0
        assert len(app.upcoming_rows) == 1
        assert app.upcoming_rows[0].title == "Polish sprint summary"

        await pilot.press("tab")
        await pilot.pause()

        await pilot.press("d")
        await pilot.pause()
        await pilot.press("y")
        for _ in range(3):
            await pilot.pause()
        assert len(app.upcoming_rows) == 0


@pytest.mark.asyncio
async def test_blank_title_shows_error(backend: FixedDayBackend) -> None:
    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        await pilot.press("a")
        await pilot.pause()
        task_input = app.screen.query_one("#task-input", Input)
        task_input.value = "   "
        await pilot.press("enter")
        for _ in range(2):
            await pilot.pause()

        assert len(app.today_rows) == 0
        assert "cannot be blank" in app.flash_message


@pytest.mark.asyncio
async def test_tab_switch_races_do_not_target_stale_selection(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-25")
    backend.add("Today task")
    upcoming = backend.add("Upcoming task")
    backend.postpone(upcoming.id)

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        app.action_next_tab()
        app.action_delete_task()
        for _ in range(4):
            await pilot.pause()

        assert app.active_tab == "upcoming"
        assert len(app.upcoming_rows) == 1
        assert app.upcoming_rows[0].title == "Upcoming task"
        assert len(app.today_rows) == 1


@pytest.mark.asyncio
async def test_switching_tabs_reconfigures_table_columns(backend: FixedDayBackend) -> None:
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        await pilot.press("tab", "tab")
        await pilot.pause()

        history_table = app.query_one("#main-table", DataTable)
        assert set(history_table.columns.keys()) == {"state", "task", "due"}

        await pilot.press("shift+tab", "shift+tab")
        await pilot.pause()

        today_table = app.query_one("#main-table", DataTable)
        assert app.active_tab == "today"
        assert set(today_table.columns.keys()) == {"task", "due"}


@pytest.mark.asyncio
async def test_action_guards_on_wrong_tab(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-25")
    backend.add("Task on today")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)

        await pilot.press("tab")
        await pilot.pause()
        assert app.active_tab == "upcoming"

        await pilot.press("x")
        await pilot.press("b")
        await pilot.press("p")
        for _ in range(3):
            await pilot.pause()

        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Task on today"


@pytest.mark.asyncio
async def test_edit_to_blank_title_shows_error(backend: FixedDayBackend) -> None:
    backend.add("Original title")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()

        await pilot.press("e")
        await pilot.pause()
        task_input = app.screen.query_one("#task-input", Input)
        task_input.value = "   "
        await pilot.press("enter")
        for _ in range(3):
            await pilot.pause()

        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Original title"
        assert "cannot be blank" in app.flash_message
