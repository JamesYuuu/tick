from __future__ import annotations

import threading
import time
from pathlib import Path

import pytest

from tick.app import TickTextualApp
from textual.widgets import DataTable
from textual.widgets import Input
from textual.widgets import Static

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


def cell_text(table: DataTable, coordinate: tuple[int, int]) -> str:
    return str(table.get_cell_at(coordinate)).strip()


def history_days_text(app: TickTextualApp) -> str:
    return " ".join(
        str(app.query_one(f"#history-day-{offset}", Static).renderable)
        for offset in range(app.HISTORY_WINDOW_DAYS)
    )


async def settle(pilot, pauses: int = 1) -> None:
    for _ in range(pauses):
        await pilot.pause()


async def open_history(pilot) -> None:
    await pilot.press("tab", "tab")
    await pilot.pause()


@pytest.fixture()
def backend(tmp_path: Path) -> FixedDayBackend:
    return FixedDayBackend(tmp_path / "todo.db", "2026-03-25")


@pytest.mark.asyncio
async def test_row_highlight_handler_is_safe_before_mount(backend: FixedDayBackend) -> None:
    app = TickTextualApp(backend)
    app._handle_row_highlighted()


@pytest.mark.asyncio
async def test_app_mounts_and_loads_snapshot(backend: FixedDayBackend) -> None:
    backend.add("Ship Python rewrite")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        assert app.snapshot_data is not None
        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Ship Python rewrite"
        assert app.active_tab == "today"
        assert app.focused == app.query_one("#main-table")
        assert str(app.query_one("#brand", Static).renderable) == "tick"
        assert str(app.query_one("#today-date", Static).renderable) == "2026-03-25 Wed"
        assert "Today" in str(app.query_one("#tab-strip", Static).renderable)
        help_text = str(app.query_one("#key-help").renderable)
        help_lines = help_text.splitlines()
        assert "task" in help_text
        assert "EDIT" in help_text
        assert "FOCUS" in help_text
        assert help_lines[0].index("b abandon") == help_lines[1].index("e edit")
        assert help_lines[1].index("e edit") == help_lines[2].index("tab switch")
        assert str(app.query_one("#status-bar", Static).renderable) == "Today shows 1 task due now."


@pytest.mark.asyncio
async def test_done_action_refreshes_today_list(backend: FixedDayBackend) -> None:
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        await pilot.press("x")
        for _ in range(3):
            await pilot.pause()
        assert len(app.today_rows) == 0
        assert str(app.query_one("#status-bar", Static).renderable) == "Completed: Finish redesign"
        table = app.query_one("#main-table", DataTable)
        assert cell_text(table, (0, 0)) == "Nothing due today."
        assert cell_text(table, (0, 1)) == "Press a in Today to add one."


@pytest.mark.asyncio
async def test_abandon_action_moves_task_into_history(backend: FixedDayBackend) -> None:
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        await pilot.press("b")
        for _ in range(3):
            await pilot.pause()

        assert len(app.today_rows) == 0
        assert str(app.query_one("#status-bar", Static).renderable) == "Abandoned: Finish redesign"

        await pilot.press("tab", "tab")
        await pilot.pause()

        assert len(app.history_rows) == 1
        assert app.history_rows[0].marker == "drop"
        assert app.history_rows[0].title == "Finish redesign"


@pytest.mark.asyncio
async def test_add_action_opens_editor_and_creates_task(backend: FixedDayBackend) -> None:
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
        assert (
            str(app.query_one("#status-bar", Static).renderable) == "Added: Write regression test"
        )


@pytest.mark.asyncio
async def test_canceling_add_edit_and_delete_keeps_state_unchanged(
    backend: FixedDayBackend,
) -> None:
    backend.add("Draft summary")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        status_before = str(app.query_one("#status-bar", Static).renderable)

        await pilot.press("a")
        await pilot.pause()
        await pilot.press("escape")
        await pilot.pause()
        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Draft summary"
        assert str(app.query_one("#status-bar", Static).renderable) == status_before

        await pilot.press("e")
        await pilot.pause()
        await pilot.press("escape")
        await pilot.pause()
        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Draft summary"
        assert str(app.query_one("#status-bar", Static).renderable) == status_before

        await pilot.press("d")
        await pilot.pause()
        await pilot.press("n")
        await pilot.pause()
        assert len(app.today_rows) == 1
        assert app.today_rows[0].title == "Draft summary"
        assert str(app.query_one("#status-bar", Static).renderable) == status_before


@pytest.mark.asyncio
async def test_cursor_navigation_updates_highlighted_row(backend: FixedDayBackend) -> None:
    for index in range(12):
        backend.add(f"Task {index}")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        table = app.query_one("#main-table", DataTable)

        # Row 0 is selected initially and keeps its plain content.
        assert table.cursor_row == 0
        assert cell_text(table, (0, 0)) == "Task 0"
        assert cell_text(table, (0, 1)) == "03-25  due today"

        # Navigate down 8 rows, then up 5.
        await pilot.press(*(["down"] * 8))
        await pilot.pause()
        await pilot.press(*(["up"] * 5))
        await pilot.pause()

        assert table.cursor_row == 3
        # Row 0 is no longer selected, so its cells revert to plain text.
        assert cell_text(table, (0, 0)) == "Task 0"
        # Row 3 is now selected.
        assert cell_text(table, (3, 0)) == "Task 3"
        # All 12 rows are still present in the table.
        assert table.row_count == 12


@pytest.mark.asyncio
async def test_refresh_with_unchanged_snapshot_preserves_table_contents(
    backend: FixedDayBackend,
) -> None:
    for index in range(4):
        backend.add(f"Task {index}")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        table = app.query_one("#main-table", DataTable)
        row_count_before = table.row_count
        cell_0_0_before = cell_text(table, (0, 0))
        cell_0_1_before = cell_text(table, (0, 1))

        await pilot.press("r", "r", "r")
        for _ in range(4):
            await pilot.pause()

        # Row count and cell contents are preserved after successive refreshes.
        assert table.row_count == row_count_before
        assert cell_text(table, (0, 0)) == cell_0_0_before
        assert cell_text(table, (0, 1)) == cell_0_1_before


@pytest.mark.asyncio
async def test_switching_tabs_moves_focus_to_active_table(backend: FixedDayBackend) -> None:
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        await pilot.press("tab")
        await pilot.pause()
        assert app.active_tab == "upcoming"
        assert app.focused == app.query_one("#main-table")
        assert "Upcoming" in str(app.query_one("#tab-strip", Static).renderable)
        help_text = str(app.query_one("#key-help").renderable)
        help_lines = help_text.splitlines()
        assert "UP NEXT" not in help_text
        assert "EDIT" in help_text
        assert "FOCUS" in help_text
        assert len(help_lines) == 2
        assert help_lines[0].index("e edit") == help_lines[1].index("↑↓ move")
        assert (
            str(app.query_one("#status-bar", Static).renderable)
            == "Upcoming empty — postponed work will land here."
        )


@pytest.mark.asyncio
async def test_switching_tabs_back_and_forth_reconfigures_shared_table(
    backend: FixedDayBackend,
) -> None:
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await pilot.pause()
        await pilot.press("tab", "tab")
        await pilot.pause()

        history_table = app.query_one("#main-table", DataTable)
        assert set(history_table.columns.keys()) == {"state", "task", "due"}
        assert app.query_one("#history-days").display is True

        await pilot.press("shift+tab", "shift+tab")
        await pilot.pause()

        today_table = app.query_one("#main-table", DataTable)
        assert app.active_tab == "today"
        assert set(today_table.columns.keys()) == {"task", "due"}
        assert app.query_one("#history-days").display is False


@pytest.mark.asyncio
async def test_tab_then_immediate_delete_does_not_target_stale_selection(tmp_path: Path) -> None:
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


@pytest.mark.asyncio
async def test_task_and_due_columns_split_available_width_evenly(backend: FixedDayBackend) -> None:
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test(size=(50, 20)) as pilot:
        await pilot.pause()
        today_table = app.query_one("#main-table", DataTable)
        assert today_table.columns["task"].width == today_table.columns["due"].width

        await pilot.press("tab")
        await pilot.pause()
        upcoming_table = app.query_one("#main-table", DataTable)
        assert upcoming_table.columns["task"].width == upcoming_table.columns["due"].width


@pytest.mark.asyncio
async def test_history_layout_uses_weighted_columns_and_aligned_help(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")

    app = TickTextualApp(backend)
    async with app.run_test(size=(84, 20)) as pilot:
        await pilot.pause()
        await pilot.press("tab", "tab")
        await pilot.pause()

        history_table = app.query_one("#main-table", DataTable)
        state_width = history_table.columns["state"].width
        task_width = history_table.columns["task"].width
        due_width = history_table.columns["due"].width
        assert state_width < due_width < task_width

        help_lines = str(app.query_one("#key-help").renderable).splitlines()
        assert len(help_lines) == 2
        assert "HISTORY" not in help_lines[0]
        assert help_lines[0].index("← back") == help_lines[1].index("↑↓ move")
        assert help_lines[0].index("→ forward") == help_lines[1].index("tab switch")


@pytest.mark.asyncio
async def test_history_columns_do_not_go_negative_in_narrow_layout(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")

    app = TickTextualApp(backend)
    async with app.run_test(size=(18, 18)) as pilot:
        await pilot.pause()
        await pilot.press("tab", "tab")
        await pilot.pause()

        history_table = app.query_one("#main-table", DataTable)
        assert history_table.columns["state"].width >= 0
        assert history_table.columns["task"].width >= 0
        assert history_table.columns["due"].width >= 0


@pytest.mark.asyncio
async def test_history_strip_moves_selection_before_scrolling_window(
    tmp_path: Path,
) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")
    backend.add("Finish redesign")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)
        assert "03-26" in history_days_text(app)
        assert (
            str(app.query_one("#status-bar", Static).renderable)
            == "History on this day is quiet — no completed or dropped tasks."
        )

        await pilot.press("left")
        await settle(pilot)
        history_days = history_days_text(app)
        assert "03-25" in history_days
        assert "03-26" in history_days
        assert (
            str(app.query_one("#status-bar", Static).renderable)
            == "History on this day is quiet — no completed or dropped tasks."
        )

        for _ in range(6):
            await pilot.press("left")
            await settle(pilot)
        history_days = history_days_text(app)
        assert "03-19" in history_days
        assert "03-26" not in history_days


@pytest.mark.asyncio
async def test_history_navigation_stops_at_today_when_pressing_future_dates(
    tmp_path: Path,
) -> None:
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
        assert "03-26" in history_days_text(app)
        assert (
            str(app.query_one("#status-bar", Static).renderable)
            == "History on this day is quiet — no completed or dropped tasks."
        )


@pytest.mark.asyncio
async def test_history_navigation_can_reach_dates_earlier_than_initial_window(
    tmp_path: Path,
) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)

        await pilot.press(*(["left"] * 10))
        await settle(pilot, 3)

        assert app.history_day == "2026-03-16"
        history_days = history_days_text(app)
        assert "03-16" in history_days
        assert "03-22" in history_days


@pytest.mark.asyncio
async def test_history_selected_day_only_counts_items_already_overdue(tmp_path: Path) -> None:
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
        assert (
            str(app.query_one("#status-bar", Static).renderable)
            == "History on this day is quiet — no completed or dropped tasks."
        )


@pytest.mark.asyncio
async def test_history_strip_hides_future_cells_near_min_boundary(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2020-01-03")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)

        assert str(app.query_one("#history-day-0", Static).renderable) == "01-01\nWed"
        assert str(app.query_one("#history-day-1", Static).renderable) == "01-02\nThu"
        assert str(app.query_one("#history-day-2", Static).renderable) == "01-03\nFri"
        assert str(app.query_one("#history-day-3", Static).renderable) == ""


@pytest.mark.asyncio
async def test_history_status_bar_includes_category_counts(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")
    done_task = backend.add("Ship release")
    dropped_task = backend.add("Archive notes")
    backend.done(done_task.id)
    backend.abandon(dropped_task.id)

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)

        assert (
            str(app.query_one("#status-bar", Static).renderable)
            == "History on this day shows 2 entries: done 1, abandoned 1, overdue 0"
        )


@pytest.mark.asyncio
async def test_history_status_bar_counts_overdue_carryovers(tmp_path: Path) -> None:
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
        assert (
            str(app.query_one("#status-bar", Static).renderable)
            == "History on this day shows 1 entry: done 0, abandoned 0, overdue 1"
        )


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


@pytest.mark.asyncio
async def test_history_quiet_to_quiet_navigation_keeps_bottom_stable(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")

    app = TickTextualApp(backend)
    async with app.run_test() as pilot:
        await settle(pilot)
        await open_history(pilot)

        status_before = str(app.query_one("#status-bar", Static).renderable)
        help_before = str(app.query_one("#key-help", Static).renderable)

        await pilot.press("left")
        assert app.flash_message == status_before
        assert str(app.query_one("#key-help", Static).renderable) == help_before

        await pilot.pause()
        assert str(app.query_one("#status-bar", Static).renderable) == status_before
        assert str(app.query_one("#key-help", Static).renderable) == help_before


@pytest.mark.asyncio
async def test_edit_delete_and_postpone_actions_keep_feedback_fresh(
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
        assert (
            str(app.query_one("#status-bar", Static).renderable) == "Updated: Polish sprint summary"
        )

        await pilot.press("p")
        for _ in range(3):
            await pilot.pause()
        assert len(app.today_rows) == 0
        assert len(app.upcoming_rows) == 1
        assert app.upcoming_rows[0].title == "Polish sprint summary"
        assert (
            str(app.query_one("#status-bar", Static).renderable)
            == "Postponed: Polish sprint summary → tomorrow"
        )

        await pilot.press("tab")
        await pilot.pause()
        assert app.active_tab == "upcoming"
        assert app.focused == app.query_one("#main-table")

        await pilot.press("d")
        await pilot.pause()
        await pilot.press("y")
        for _ in range(3):
            await pilot.pause()
        assert len(app.upcoming_rows) == 0
        assert (
            str(app.query_one("#status-bar", Static).renderable) == "Deleted: Polish sprint summary"
        )
        table = app.query_one("#main-table", DataTable)
        assert cell_text(table, (0, 0)) == "No upcoming tasks."
        assert cell_text(table, (0, 1)) == "Press a in Today to add one."


@pytest.mark.asyncio
async def test_blank_title_shows_inline_error_feedback(backend: FixedDayBackend) -> None:
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
        assert str(app.query_one("#status-bar", Static).renderable) == "title cannot be blank"
        table = app.query_one("#main-table", DataTable)
        assert cell_text(table, (0, 0)) == "Nothing due today."
        assert cell_text(table, (0, 1)) == "Press a in Today to add one."
