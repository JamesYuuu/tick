from __future__ import annotations

import asyncio
from dataclasses import dataclass
from datetime import date, timedelta

from textual import on
from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Container, Horizontal, Vertical
from textual.screen import ModalScreen
from textual.widgets import DataTable, Footer, Header, Input, Label, Static, TabbedContent, TabPane

from .backend import BackendError, Snapshot, Task, TickBackend


def parse_day(value: str) -> date:
    return date.fromisoformat(value)


def format_mmdd(value: str) -> str:
    return parse_day(value).strftime("%m-%d")


def format_long_day(value: str) -> str:
    return parse_day(value).strftime("%A, %b %d")


class TaskEditorScreen(ModalScreen[str | None]):
    def __init__(self, title: str, initial_value: str = "") -> None:
        super().__init__()
        self.screen_title = title
        self.initial_value = initial_value

    def compose(self) -> ComposeResult:
        with Container(id="modal-shell"):
            yield Label(self.screen_title, id="modal-title")
            yield Input(value=self.initial_value, placeholder="Task title", id="task-input")
            yield Label("Enter submit, Esc cancel", id="modal-help")

    def on_mount(self) -> None:
        self.query_one(Input).focus()

    def on_input_submitted(self, event: Input.Submitted) -> None:
        self.dismiss(event.value)

    def key_escape(self) -> None:
        self.dismiss(None)


class ConfirmScreen(ModalScreen[bool]):
    def __init__(self, title: str, body: str) -> None:
        super().__init__()
        self.screen_title = title
        self.body = body

    def compose(self) -> ComposeResult:
        with Container(id="modal-shell", classes="confirm"):
            yield Label(self.screen_title, id="modal-title")
            yield Static(self.body, id="confirm-body")
            yield Label("y confirm, n / Esc cancel", id="modal-help")

    def key_y(self) -> None:
        self.dismiss(True)

    def key_n(self) -> None:
        self.dismiss(False)

    def key_escape(self) -> None:
        self.dismiss(False)


@dataclass
class HistoryRow:
    marker: str
    title: str
    due_day: str
    tone: str
    context: str


class TickTextualApp(App[None]):
    CSS_PATH = "tick.tcss"
    BINDINGS = [
        Binding("q", "quit", "Quit"),
        Binding("tab", "next_tab", "Next Tab"),
        Binding("shift+tab", "previous_tab", "Prev Tab"),
        Binding("a", "add_task", "Add", show=False),
        Binding("e", "edit_task", "Edit", show=False),
        Binding("d", "delete_task", "Delete", show=False),
        Binding("x", "done_task", "Done", show=False),
        Binding("b", "abandon_task", "Abandon", show=False),
        Binding("p", "postpone_task", "Postpone", show=False),
        Binding("left,h", "history_prev_day", "Prev Day", show=False),
        Binding("right,l", "history_next_day", "Next Day", show=False),
        Binding("r", "refresh", "Refresh"),
    ]

    def __init__(self, backend: TickBackend) -> None:
        super().__init__()
        self.backend = backend
        self.snapshot_data: Snapshot | None = None
        self.history_day: str | None = None
        self.today_rows: list[Task] = []
        self.upcoming_rows: list[Task] = []
        self.history_rows: list[HistoryRow] = []
        self.flash_message = "Loading workspace"

    def compose(self) -> ComposeResult:
        yield Header(show_clock=True)
        yield Static(id="hero")
        with TabbedContent(id="views"):
            with TabPane("Today", id="today"):
                with Horizontal(classes="pane-shell"):
                    with Vertical(classes="content-rail"):
                        yield Static(id="today-summary", classes="summary")
                        yield DataTable(id="today-table", classes="task-table", cursor_type="row")
                    with Vertical(classes="detail-rail"):
                        yield Static(id="today-focus", classes="focus-card")
                        yield Static(id="today-actions", classes="hint-card")
            with TabPane("Upcoming", id="upcoming"):
                with Horizontal(classes="pane-shell"):
                    with Vertical(classes="content-rail"):
                        yield Static(id="upcoming-summary", classes="summary")
                        yield DataTable(id="upcoming-table", classes="task-table", cursor_type="row")
                    with Vertical(classes="detail-rail"):
                        yield Static(id="upcoming-focus", classes="focus-card")
                        yield Static(id="upcoming-actions", classes="hint-card")
            with TabPane("History", id="history"):
                with Horizontal(classes="pane-shell"):
                    with Vertical(classes="content-rail"):
                        yield Static(id="history-days", classes="history-days")
                        with Horizontal(id="history-stats"):
                            yield Static(id="history-done-stat", classes="stat-card")
                            yield Static(id="history-abandoned-stat", classes="stat-card")
                            yield Static(id="history-overdue-stat", classes="stat-card")
                        yield DataTable(id="history-table", classes="task-table", cursor_type="row")
                    with Vertical(classes="detail-rail"):
                        yield Static(id="history-focus", classes="focus-card")
                        yield Static(id="history-actions", classes="hint-card")
        yield Static(id="flash")
        yield Footer()

    def on_mount(self) -> None:
        self._prepare_tables()
        self.run_worker(self._load_snapshot(), exclusive=True, group="snapshot")

    def _prepare_tables(self) -> None:
        today_table = self.query_one("#today-table", DataTable)
        today_table.add_columns("Task", "Due")
        upcoming_table = self.query_one("#upcoming-table", DataTable)
        upcoming_table.add_columns("Task", "Due")
        history_table = self.query_one("#history-table", DataTable)
        history_table.add_columns("State", "Task", "Due")

    async def _load_snapshot(self) -> None:
        try:
            snapshot = await asyncio.to_thread(self.backend.snapshot, self.history_day)
        except BackendError as exc:
            self.notify(str(exc), severity="error")
            return
        self._apply_snapshot(snapshot)

    def _apply_snapshot(self, snapshot: Snapshot) -> None:
        self.snapshot_data = snapshot
        self.history_day = snapshot.history.selected_day
        self.today_rows = snapshot.today
        self.upcoming_rows = snapshot.upcoming
        self.history_rows = self._build_history_rows(snapshot)

        self._fill_task_table(
            self.query_one("#today-table", DataTable),
            self.today_rows,
            empty_text="Nothing due today.",
        )
        self._fill_task_table(
            self.query_one("#upcoming-table", DataTable),
            self.upcoming_rows,
            empty_text="No upcoming tasks.",
        )
        self._fill_history_table(self.query_one("#history-table", DataTable), self.history_rows)
        self._render_summaries(snapshot)
        self._refresh_detail_panels()
        self.sub_title = f"Today {snapshot.current_day}"
        self._set_flash(f"Synced {snapshot.current_day}")

    def _render_summaries(self, snapshot: Snapshot) -> None:
        overdue_today = sum(1 for task in snapshot.today if task.due_day < snapshot.current_day)
        streak = "Clear runway" if not overdue_today else "Needs rescue"
        self.query_one("#hero", Static).update(
            "\n".join(
                [
                    f"tick  {format_long_day(snapshot.current_day)}",
                    f"{len(snapshot.today)} active now  |  {len(snapshot.upcoming)} queued next  |  {streak}",
                ]
            )
        )
        self.query_one("#today-summary", Static).update(
            "\n".join(
                [
                    "Today Board",
                    f"{len(snapshot.today)} active tasks",
                    f"{overdue_today} overdue and asking for attention",
                ]
            )
        )
        self.query_one("#upcoming-summary", Static).update(
            "\n".join(
                [
                    "Upcoming Queue",
                    f"{len(snapshot.upcoming)} tasks scheduled after today",
                    "Use this lane to shape what tomorrow looks like",
                ]
            )
        )

        history = snapshot.history
        self.query_one("#history-days", Static).update(self._render_history_days(history.selected_day))
        self.query_one("#history-done-stat", Static).update(
            "\n".join(
                [
                    "Done",
                    f"{len(history.done)} finished on {format_mmdd(history.selected_day)}",
                    f"Delayed {history.stats.delayed_done}  |  Rate {history.stats.done_delayed_ratio:.0%}",
                ]
            )
        )
        self.query_one("#history-abandoned-stat", Static).update(
            "\n".join(
                [
                    "Abandoned",
                    f"{len(history.abandoned)} dropped on {format_mmdd(history.selected_day)}",
                    f"Delayed {history.stats.delayed_abandoned}  |  Rate {history.stats.abandoned_delayed_ratio:.0%}",
                ]
            )
        )
        overdue_active = sum(
            1 for task in history.active_created if task.status == "active" and task.due_day < snapshot.current_day
        )
        self.query_one("#history-overdue-stat", Static).update(
            "\n".join(
                [
                    "Overdue Active",
                    f"{overdue_active} tasks born on the selected day are still late",
                    f"Window {format_mmdd(history.from_day)} to {format_mmdd(history.to_day)}",
                ]
            )
        )

    def _render_history_days(self, selected_day: str) -> str:
        end = parse_day(selected_day)
        labels: list[str] = []
        for offset in range(6, -1, -1):
            candidate = end - timedelta(days=offset)
            token = candidate.strftime("%m-%d")
            if candidate.isoformat() == selected_day:
                labels.append(f"[ {token} ]")
            else:
                labels.append(token)
        return "  ".join(labels)

    def _fill_task_table(self, table: DataTable, rows: list[Task], empty_text: str) -> None:
        previous = table.cursor_row if table.row_count else 0
        table.clear(columns=False)
        if not rows:
            table.add_row(empty_text, "", height=1)
            table.cursor_type = "none"
            return
        table.cursor_type = "row"
        for task in rows:
            table.add_row(task.title, self._task_due_label(task), height=1)
        table.move_cursor(row=min(previous, len(rows) - 1), animate=False)

    def _fill_history_table(self, table: DataTable, rows: list[HistoryRow]) -> None:
        previous = table.cursor_row if table.row_count else 0
        table.clear(columns=False)
        if not rows:
            table.add_row(" ", "No history tasks.", "", height=1)
            table.cursor_type = "none"
            return
        table.cursor_type = "row"
        for row in rows:
            table.add_row(row.marker, row.title, f"{format_mmdd(row.due_day)}  {row.context}", height=1)
        table.move_cursor(row=min(previous, len(rows) - 1), animate=False)

    def _build_history_rows(self, snapshot: Snapshot) -> list[HistoryRow]:
        rows: list[HistoryRow] = []
        for task in snapshot.history.done:
            rows.append(
                HistoryRow(marker="done", title=task.title, due_day=task.due_day, tone="complete", context="finished")
            )
        for task in snapshot.history.abandoned:
            rows.append(
                HistoryRow(marker="drop", title=task.title, due_day=task.due_day, tone="abandoned", context="dropped")
            )
        for task in snapshot.history.active_created:
            if task.status == "active" and task.due_day < snapshot.current_day:
                rows.append(
                    HistoryRow(marker="late", title=task.title, due_day=task.due_day, tone="late", context="still open")
                )
        return rows

    def _task_due_label(self, task: Task) -> str:
        if self.snapshot_data is None:
            return format_mmdd(task.due_day)
        delta = (parse_day(task.due_day) - parse_day(self.snapshot_data.current_day)).days
        if delta < 0:
            return f"{format_mmdd(task.due_day)}  overdue by {abs(delta)}d"
        if delta == 0:
            return f"{format_mmdd(task.due_day)}  due today"
        return f"{format_mmdd(task.due_day)}  in {delta}d"

    def _set_flash(self, message: str) -> None:
        self.flash_message = message
        self.query_one("#flash", Static).update(message)

    def _refresh_detail_panels(self) -> None:
        self._update_task_focus(
            "today",
            self._selected_from_table("#today-table", self.today_rows),
            "Complete, postpone, or clean up the work in front of you.",
        )
        self._update_task_focus(
            "upcoming",
            self._selected_from_table("#upcoming-table", self.upcoming_rows),
            "Shape upcoming work before it becomes pressure.",
        )
        self._update_history_focus()
        self.query_one("#today-actions", Static).update(
            "Today Keys\n[a] add\n[e] edit\n[d] delete\n[x] done\n[b] abandon\n[p] postpone"
        )
        self.query_one("#upcoming-actions", Static).update(
            "Upcoming Keys\n[e] edit\n[d] delete\n[tab] switch view\n[r] refresh"
        )
        self.query_one("#history-actions", Static).update(
            "History Keys\n[h/left] previous day\n[l/right] next day\n[tab] switch view\n[r] refresh"
        )

    def _update_task_focus(self, prefix: str, task: Task | None, empty_message: str) -> None:
        widget = self.query_one(f"#{prefix}-focus", Static)
        if task is None:
            widget.update("\n".join(["Focus", "No task selected", empty_message]))
            return
        widget.update(
            "\n".join(
                [
                    "Focus",
                    task.title,
                    f"ID {task.id}  |  Created {format_mmdd(task.created_day)}",
                    f"Due {self._task_due_label(task)}",
                ]
            )
        )

    def _update_history_focus(self) -> None:
        widget = self.query_one("#history-focus", Static)
        if self.snapshot_data is None:
            widget.update("Focus\nHistory is loading")
            return
        if not self.history_rows:
            widget.update(
                "\n".join(
                    [
                        "Focus",
                        f"No archived movement on {format_long_day(self.snapshot_data.history.selected_day)}",
                        "Try moving through the date strip to inspect another day.",
                    ]
                )
            )
            return
        table = self.query_one("#history-table", DataTable)
        row_index = max(0, min(table.cursor_row, len(self.history_rows) - 1))
        row = self.history_rows[row_index]
        widget.update(
            "\n".join(
                [
                    "Focus",
                    row.title,
                    f"State {row.marker}  |  Due {format_mmdd(row.due_day)}",
                    f"Status note: {row.context}",
                ]
            )
        )

    def action_next_tab(self) -> None:
        tabs = self.query_one(TabbedContent)
        order = ["today", "upcoming", "history"]
        current = tabs.active or "today"
        tabs.active = order[(order.index(current) + 1) % len(order)]
        self._refresh_detail_panels()

    def action_previous_tab(self) -> None:
        tabs = self.query_one(TabbedContent)
        order = ["today", "upcoming", "history"]
        current = tabs.active or "today"
        tabs.active = order[(order.index(current) - 1) % len(order)]
        self._refresh_detail_panels()

    def action_refresh(self) -> None:
        self._set_flash("Refreshing snapshot")
        self.run_worker(self._load_snapshot(), exclusive=True, group="snapshot")

    @property
    def active_tab(self) -> str:
        return self.query_one(TabbedContent).active or "today"

    def _selected_task(self) -> Task | None:
        active = self.active_tab
        if active == "today":
            return self._selected_from_table("#today-table", self.today_rows)
        if active == "upcoming":
            return self._selected_from_table("#upcoming-table", self.upcoming_rows)
        return None

    def _selected_from_table(self, selector: str, rows: list[Task]) -> Task | None:
        if not rows:
            return None
        table = self.query_one(selector, DataTable)
        row = table.cursor_row
        if row < 0 or row >= len(rows):
            return None
        return rows[row]

    async def action_add_task(self) -> None:
        if self.active_tab != "today":
            return
        value = await self.push_screen_wait(TaskEditorScreen("Add Task"))
        if value is None:
            return
        try:
            await asyncio.to_thread(self.backend.add, value)
        except BackendError as exc:
            self._set_flash(str(exc))
            self.notify(str(exc), severity="error")
            return
        self._set_flash("Task added")
        self.notify("Task added")
        self.action_refresh()

    async def action_edit_task(self) -> None:
        task = self._selected_task()
        if task is None:
            return
        value = await self.push_screen_wait(TaskEditorScreen("Edit Task", task.title))
        if value is None:
            return
        try:
            await asyncio.to_thread(self.backend.edit, task.id, value)
        except BackendError as exc:
            self._set_flash(str(exc))
            self.notify(str(exc), severity="error")
            return
        self._set_flash("Task updated")
        self.notify("Task updated")
        self.action_refresh()

    async def action_delete_task(self) -> None:
        task = self._selected_task()
        if task is None:
            return
        confirmed = await self.push_screen_wait(ConfirmScreen("Delete Task", task.title))
        if not confirmed:
            return
        try:
            await asyncio.to_thread(self.backend.delete, task.id)
        except BackendError as exc:
            self._set_flash(str(exc))
            self.notify(str(exc), severity="error")
            return
        self._set_flash("Task deleted")
        self.notify("Task deleted")
        self.action_refresh()

    async def action_done_task(self) -> None:
        if self.active_tab != "today":
            return
        task = self._selected_task()
        if task is None:
            return
        try:
            await asyncio.to_thread(self.backend.done, task.id)
        except BackendError as exc:
            self._set_flash(str(exc))
            self.notify(str(exc), severity="error")
            return
        self._set_flash("Task completed")
        self.notify("Task completed")
        self.action_refresh()

    async def action_abandon_task(self) -> None:
        if self.active_tab != "today":
            return
        task = self._selected_task()
        if task is None:
            return
        try:
            await asyncio.to_thread(self.backend.abandon, task.id)
        except BackendError as exc:
            self._set_flash(str(exc))
            self.notify(str(exc), severity="error")
            return
        self._set_flash("Task abandoned")
        self.notify("Task abandoned")
        self.action_refresh()

    async def action_postpone_task(self) -> None:
        if self.active_tab != "today":
            return
        task = self._selected_task()
        if task is None:
            return
        try:
            await asyncio.to_thread(self.backend.postpone, task.id)
        except BackendError as exc:
            self._set_flash(str(exc))
            self.notify(str(exc), severity="error")
            return
        self._set_flash("Task postponed by one day")
        self.notify("Task postponed")
        self.action_refresh()

    def action_history_prev_day(self) -> None:
        if self.active_tab != "history" or self.history_day is None:
            return
        self.history_day = (parse_day(self.history_day) - timedelta(days=1)).isoformat()
        self._set_flash(f"History moved to {self.history_day}")
        self.action_refresh()

    def action_history_next_day(self) -> None:
        if self.active_tab != "history" or self.history_day is None or self.snapshot_data is None:
            return
        current_day = parse_day(self.snapshot_data.current_day)
        next_day = parse_day(self.history_day) + timedelta(days=1)
        if next_day > current_day:
            return
        self.history_day = next_day.isoformat()
        self._set_flash(f"History moved to {self.history_day}")
        self.action_refresh()

    @on(TabbedContent.TabActivated)
    def _handle_tab_change(self) -> None:
        if self.active_tab == "history" and self.history_day is None and self.snapshot_data is not None:
            self.history_day = self.snapshot_data.current_day
        self._refresh_detail_panels()

    @on(DataTable.RowHighlighted)
    def _handle_row_highlighted(self) -> None:
        self._refresh_detail_panels()
