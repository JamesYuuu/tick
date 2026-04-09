from __future__ import annotations

import asyncio
from collections import deque
from collections.abc import Sequence
from dataclasses import dataclass
from datetime import date, timedelta
from typing import Callable, Literal

from textual import events, on
from textual.app import App, ComposeResult
from textual.binding import Binding
from textual.containers import Container, Horizontal
from textual.css.query import NoMatches
from textual.screen import ModalScreen
from textual.widgets import DataTable, Input, Label, Static
from rich.text import Text

from .backend import BackendError, Snapshot, Task, TickBackend


Severity = Literal["information", "warning", "error"]


def parse_day(value: str) -> date:
    return date.fromisoformat(value)


def format_mmdd(value: str) -> str:
    return parse_day(value).strftime("%m-%d")


def format_top_bar_day(value: str) -> str:
    return parse_day(value).strftime("%Y-%m-%d %a")


def normalize_title(value: str) -> str:
    return " ".join(value.split())


def weighted_widths(total_width: int, weights: Sequence[int]) -> tuple[int, ...]:
    if total_width <= 0:
        return tuple(0 for _ in weights)
    total_weight = sum(weights)
    widths = [(total_width * weight) // total_weight for weight in weights]
    remainders = [(total_width * weight) % total_weight for weight in weights]
    remaining = total_width - sum(widths)
    for index in sorted(range(len(weights)), key=lambda i: remainders[i], reverse=True)[:remaining]:
        widths[index] += 1
    while any(width == 0 for width in widths) and any(width > 1 for width in widths):
        smallest = widths.index(0)
        largest = max(range(len(widths)), key=lambda i: widths[i])
        widths[largest] -= 1
        widths[smallest] += 1
    return tuple(widths)


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
    context: str


class TickTextualApp(App[None]):
    CSS_PATH = "tick.tcss"
    HISTORY_WINDOW_DAYS = 7
    HISTORY_STATE_COLUMN_WIDTH = 8
    HELP_LABEL_WIDTH = 9
    HELP_ITEM_WIDTH = 13
    TABLE_ROW_HEIGHT = 3
    TABLE_HEADER_HEIGHT = 3
    BRAND_LABEL = "tick"
    TAB_ORDER = ("today", "upcoming", "history")
    TAB_LABELS = {"today": "Today", "upcoming": "Upcoming", "history": "History"}
    TASK_TABLES = {"today": "today_rows", "upcoming": "upcoming_rows"}
    HELP_ROWS = {
        "today": (
            ("task", "x done", "b abandon", "p postpone"),
            ("EDIT", "a add", "e edit", "d delete"),
            ("FOCUS", "↑↓ move", "tab switch", "r refresh", "q quit"),
        ),
        "upcoming": (
            ("EDIT", "e edit", "d delete"),
            ("FOCUS", "↑↓ move", "tab switch", "r refresh", "q quit"),
        ),
        "history": (
            ("DATE", "← back", "→ forward"),
            ("FOCUS", "↑↓ move", "tab switch", "r refresh", "q quit"),
        ),
    }
    BINDINGS = [
        Binding("q", "quit", "Quit"),
        Binding("tab", "next_tab", "Next Tab", priority=True),
        Binding("shift+tab", "previous_tab", "Prev Tab", priority=True),
        Binding("a", "add_task", "Add", show=False),
        Binding("e", "edit_task", "Edit", show=False),
        Binding("d", "delete_task", "Delete", show=False),
        Binding("x", "done_task", "Done", show=False),
        Binding("b", "abandon_task", "Abandon", show=False),
        Binding("p", "postpone_task", "Postpone", show=False),
        Binding("left", "history_prev_day", "Prev Day", show=False, priority=True),
        Binding("right", "history_next_day", "Next Day", show=False, priority=True),
        Binding("r", "refresh", "Refresh"),
    ]

    def __init__(self, backend: TickBackend) -> None:
        super().__init__()
        self.backend = backend
        self.snapshot_data: Snapshot | None = None
        self.current_tab = "today"
        self.history_day: str | None = None
        self.history_strip_start: str | None = None
        self.current_day_label = format_top_bar_day(self.backend.current_day())
        self.today_rows: list[Task] = []
        self.upcoming_rows: list[Task] = []
        self.history_rows: list[HistoryRow] = []
        self.flash_message = "Loading workspace…"
        self._rendered_key_help: str | None = None
        self._rendered_status_bar: str | None = None
        self._render_scheduled = False
        self._tables_dirty = False
        self._configured_tab: str | None = None
        self._snapshot_loading = False
        self._snapshot_request_id = 0
        self._snapshot_processed_id = 0
        self._snapshot_waiters: dict[int, list[asyncio.Future[None]]] = {}
        self._mutation_lock = asyncio.Lock()
        self._pending_flash_messages: deque[str] = deque()
        self._selected_rows: dict[str, int | None] = {tab: None for tab in self.TAB_ORDER}

    def compose(self) -> ComposeResult:
        with Container(id="app-shell"):
            with Horizontal(id="top-bar"):
                yield Static(self.BRAND_LABEL, id="brand")
                yield Static(id="tab-strip")
                yield Static(self.current_day_label, id="today-date")
            with Container(id="main-pane", classes="pane-frame"):
                with Horizontal(id="history-days", classes="history-strip"):
                    for offset in range(self.HISTORY_WINDOW_DAYS):
                        yield Static(id=f"history-day-{offset}", classes="history-day")
                yield DataTable(
                    id="main-table",
                    classes="task-table",
                    cursor_type="row",
                    header_height=self.TABLE_HEADER_HEIGHT,
                )
            yield Static(id="status-bar")
            yield Static(id="key-help")

    def on_mount(self) -> None:
        self._prepare_table()
        self._schedule_render()
        self._request_snapshot()

    def _start_snapshot_worker(self) -> None:
        if self._snapshot_loading:
            return
        self._snapshot_loading = True
        self.run_worker(self._drain_snapshot_requests(), group="snapshot")

    def _enqueue_snapshot_request(self) -> int:
        self._snapshot_request_id += 1
        self._start_snapshot_worker()
        return self._snapshot_request_id

    def _prepare_table(self) -> None:
        self._table().show_horizontal_scrollbar = False

    def _request_snapshot(self) -> None:
        self._enqueue_snapshot_request()

    def _request_snapshot_future(self) -> asyncio.Future[None]:
        request_id = self._enqueue_snapshot_request()
        future: asyncio.Future[None] = asyncio.get_running_loop().create_future()
        self._snapshot_waiters.setdefault(request_id, []).append(future)
        return future

    async def _drain_snapshot_requests(self) -> None:
        try:
            while self._snapshot_processed_id < self._snapshot_request_id:
                target_request_id = self._snapshot_request_id
                try:
                    snapshot = await asyncio.to_thread(self.backend.snapshot, self.history_day)
                except BackendError as exc:
                    self._show_feedback(str(exc), severity="error")
                else:
                    self._apply_snapshot(snapshot)
                self._snapshot_processed_id = target_request_id
                self._resolve_snapshot_waiters()
        finally:
            self._snapshot_loading = False
            if self._snapshot_processed_id < self._snapshot_request_id:
                self._start_snapshot_worker()

    def _resolve_snapshot_waiters(self) -> None:
        ready = [
            request_id
            for request_id in self._snapshot_waiters
            if request_id <= self._snapshot_processed_id
        ]
        for request_id in sorted(ready):
            for future in self._snapshot_waiters.pop(request_id):
                if not future.done():
                    future.set_result(None)

    def _apply_snapshot(self, snapshot: Snapshot) -> None:
        self.snapshot_data = snapshot
        self.current_day_label = format_top_bar_day(snapshot.current_day)
        self.history_day = snapshot.history.selected_day
        self._coerce_history_strip_window(snapshot.current_day)
        self.today_rows = snapshot.today
        self.upcoming_rows = snapshot.upcoming
        self.history_rows = self._build_history_rows(snapshot)
        self._tables_dirty = True
        self.sub_title = f"Today {snapshot.current_day}"
        if self._pending_flash_messages:
            self.flash_message = self._pending_flash_messages.pop()
            self._pending_flash_messages.clear()
        else:
            self.flash_message = self._status_summary(snapshot)
        self._schedule_render()

    def _status_summary(self, snapshot: Snapshot | None = None) -> str:
        snapshot = snapshot or self.snapshot_data
        if snapshot is None:
            return self.flash_message
        tab = self.active_tab
        if tab == "today":
            n = len(self.today_rows)
            return (
                "Today clear — press a to capture a new task."
                if n == 0
                else f"Today shows {n} task{'s' if n != 1 else ''} due now."
            )
        if tab == "upcoming":
            n = len(self.upcoming_rows)
            return (
                "Upcoming empty — postponed work will land here."
                if n == 0
                else f"Upcoming tracks {n} scheduled task{'s' if n != 1 else ''}."
            )
        n = len(self.history_rows)
        if n == 0:
            return "History on this day is quiet — no completed or dropped tasks."
        done_count = len(snapshot.history.done)
        abandoned_count = len(snapshot.history.abandoned)
        overdue_count = len(snapshot.history.active_due)
        return (
            f"History on this day shows {n} entr{'y' if n == 1 else 'ies'}: "
            f"done {done_count}, abandoned {abandoned_count}, overdue {overdue_count}"
        )

    def _set_flash(self, message: str) -> None:
        if self.flash_message == message:
            return
        self.flash_message = message
        if self.is_mounted:  # type: ignore[truthy-function]
            self._schedule_render()

    def _show_feedback(self, message: str, *, severity: Severity = "information") -> None:
        self._set_flash(message)
        self.notify(message, severity=severity)

    def _schedule_render(self) -> None:
        if self._render_scheduled:
            return
        self._render_scheduled = True
        self.call_after_refresh(self._flush_render)

    def _flush_render(self) -> None:
        self._render_scheduled = False
        table = self._table()
        self._configure_table(table)
        if self._tables_dirty:
            self._fill_table(
                table,
                self._active_rows(),
                self._render_active_row,
                self._empty_row(),
                self._selected_rows[self.active_tab],
            )
            self._tables_dirty = False
        self._balance_table_columns(table)
        self._render_history_strip()
        self._render_top_bar()
        self._render_status_bar()
        self._render_key_help()
        self._focus_table()

    def _balance_task_due_columns(self, table: DataTable) -> None:
        if table.size.width <= 0:
            return
        padding = 2 * table.cell_padding
        available_width = max(2, table.size.width - (2 * padding))
        task_width, due_width = weighted_widths(available_width, (1, 1))
        table.columns["task"].auto_width = False  # type: ignore[index]
        table.columns["task"].width = task_width  # type: ignore[index]
        table.columns["due"].auto_width = False  # type: ignore[index]
        table.columns["due"].width = due_width  # type: ignore[index]
        table.refresh(layout=True)

    def _balance_history_columns(self, table: DataTable) -> None:
        if table.size.width <= 0:
            return
        padding = 2 * table.cell_padding
        available_width = max(3, table.size.width - (3 * padding))
        state_width, task_width, due_width = weighted_widths(available_width, (1, 4, 2))
        table.columns["state"].auto_width = False  # type: ignore[index]
        table.columns["state"].width = state_width  # type: ignore[index]
        table.columns["task"].auto_width = False  # type: ignore[index]
        table.columns["task"].width = task_width  # type: ignore[index]
        table.columns["due"].auto_width = False  # type: ignore[index]
        table.columns["due"].width = due_width  # type: ignore[index]
        table.refresh(layout=True)

    def _balance_table_columns(self, table: DataTable) -> None:
        if self.active_tab == "history":
            self._balance_history_columns(table)
        else:
            self._balance_task_due_columns(table)

    def _render_history_strip(self) -> None:
        strip = self.query_one("#history-days", Horizontal)
        if self.active_tab == "history" and self.snapshot_data is not None:
            strip.display = True
            self._update_history_strip_cells()
        else:
            strip.display = False

    def _table(self) -> DataTable:
        return self.query_one("#main-table", DataTable)

    def _focus_table(self) -> None:
        try:
            self.set_focus(self._table())
        except NoMatches:
            pass

    def _coerce_history_strip_window(self, current_day_value: str) -> None:
        current_day = parse_day(current_day_value)
        selected_day = parse_day(self.history_day or current_day_value)
        max_start = current_day - timedelta(days=self.HISTORY_WINDOW_DAYS - 1)
        start = (
            parse_day(self.history_strip_start)
            if self.history_strip_start is not None
            else max_start
        )
        window_end = start + timedelta(days=self.HISTORY_WINDOW_DAYS - 1)
        if selected_day < start:
            start = selected_day
        elif selected_day > window_end:
            start = selected_day - timedelta(days=self.HISTORY_WINDOW_DAYS - 1)
        if start > max_start:
            start = max_start
        if start < self.backend.MIN_HISTORY_DATE:
            start = self.backend.MIN_HISTORY_DATE
        self.history_strip_start = start.isoformat()

    def _history_strip_cells(self) -> list[tuple[str, str, bool]]:
        if (
            self.history_strip_start is None
            or self.history_day is None
            or self.snapshot_data is None
        ):
            return []
        start = parse_day(self.history_strip_start)
        current_day = parse_day(self.snapshot_data.current_day)
        selected_day = self.history_day
        cells: list[tuple[str, str, bool]] = []
        for offset in range(self.HISTORY_WINDOW_DAYS):
            candidate = start + timedelta(days=offset)
            if candidate > current_day:
                cells.append(("", "", False))
                continue
            cells.append(
                (
                    candidate.strftime("%m-%d"),
                    candidate.strftime("%a"),
                    candidate.isoformat() == selected_day,
                )
            )
        return cells

    def _update_history_strip_cells(self) -> None:
        cells = self._history_strip_cells()
        for offset, (date_token, weekday_token, selected) in enumerate(cells):
            cell = self.query_one(f"#history-day-{offset}", Static)
            cell.update(f"{date_token}\n{weekday_token}".strip())
            if selected:
                cell.add_class("is-selected")
            else:
                cell.remove_class("is-selected")

    def _configure_table(self, table: DataTable) -> None:
        if self._configured_tab == self.active_tab:
            return
        table.clear(columns=True)
        for label, key, width in self._column_specs():
            if width is None:
                table.add_column(self._padded_text(label), key=key)
            else:
                table.add_column(self._padded_text(label), key=key, width=width)
        self._configured_tab = self.active_tab
        self._tables_dirty = True

    def _column_specs(self) -> tuple[tuple[str, str, int | None], ...]:
        if self.active_tab == "history":
            return (
                ("State", "state", self.HISTORY_STATE_COLUMN_WIDTH),
                ("Task", "task", None),
                ("Due", "due", None),
            )
        return (("Task", "task", None), ("Due", "due", None))

    def _active_rows(self) -> Sequence[object]:
        if self.active_tab == "today":
            return self.today_rows
        if self.active_tab == "upcoming":
            return self.upcoming_rows
        return self.history_rows

    def _render_active_row(self, row: object) -> tuple[str, ...]:
        if isinstance(row, HistoryRow):
            return self._history_cells(row)
        if not isinstance(row, Task):
            raise TypeError(f"unsupported row type: {type(row)!r}")
        return self._task_cells(row)

    def _empty_row(self) -> tuple[str, ...]:
        if self.active_tab == "today":
            return (
                self._padded_text("Nothing due today."),
                self._padded_text("Press a in Today to add one."),
            )
        if self.active_tab == "upcoming":
            return (
                self._padded_text("No upcoming tasks."),
                self._padded_text("Press a in Today to add one."),
            )
        return (
            self._padded_text("·"),
            self._padded_text("No history updates for this day."),
            self._padded_text("Try ← / → to browse."),
        )

    def _padded_text(self, value: str) -> str:
        return f"\n{value}\n"

    def _fill_table(
        self,
        table: DataTable,
        rows: Sequence[object],
        render_row: Callable[[object], tuple[str, ...]],
        empty_row: tuple[str, ...],
        selected_row: int | None = None,
    ) -> None:
        previous = selected_row if selected_row is not None else 0
        table.clear(columns=False)
        if not rows:
            table.add_row(*empty_row, height=self.TABLE_ROW_HEIGHT)
            table.cursor_type = "none"
            self._selected_rows[self.active_tab] = None
            return
        table.cursor_type = "row"
        active_row = min(previous, len(rows) - 1)
        for row in rows:
            table.add_row(*render_row(row), height=self.TABLE_ROW_HEIGHT)
        table.move_cursor(row=active_row, animate=False)
        self._selected_rows[self.active_tab] = active_row

    def _build_history_rows(self, snapshot: Snapshot) -> list[HistoryRow]:
        rows: list[HistoryRow] = []
        for t in snapshot.history.done:
            rows.append(HistoryRow("done", t.title, t.due_day, "finished"))
        for t in snapshot.history.abandoned:
            rows.append(HistoryRow("drop", t.title, t.due_day, "dropped"))
        for t in snapshot.history.active_due:
            rows.append(HistoryRow("late", t.title, t.due_day, "still open"))
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

    def _task_cells(self, task: Task) -> tuple[str, str]:
        due = self._task_due_label(task)
        return (self._padded_text(task.title), self._padded_text(due))

    def _history_cells(self, row: HistoryRow) -> tuple[str, str, str]:
        due = f"{format_mmdd(row.due_day)}  {row.context}"
        return (
            self._padded_text(row.marker),
            self._padded_text(row.title),
            self._padded_text(due),
        )

    def _render_top_bar(self) -> None:
        tab_strip = self.query_one("#tab-strip", Static)
        today_date = self.query_one("#today-date", Static)
        tabs = Text()
        for tab_id, label in self.TAB_LABELS.items():
            if tabs:
                tabs.append("  ")
            style = "bold #f5f7ff on #5b6fcf" if tab_id == self.active_tab else "bold #d7dcf7"
            tabs.append(f" {label} ", style=style)
        tab_strip.update(tabs)
        today_date.update(self.current_day_label)

    def _render_key_help(self) -> None:
        def row(label: str, *items: str) -> str:
            return (
                f"{label:<{self.HELP_LABEL_WIDTH}}"
                + "".join(f"{item:<{self.HELP_ITEM_WIDTH}}" for item in items)
            ).rstrip()

        text = "\n".join(row(*items) for items in self.HELP_ROWS[self.active_tab])
        if text == self._rendered_key_help:
            return
        self._rendered_key_help = text
        self.query_one("#key-help", Static).update(text)

    def _render_status_bar(self, message: str | None = None) -> None:
        text = self.flash_message if message is None else message
        if text == self._rendered_status_bar:
            return
        self._rendered_status_bar = text
        self.query_one("#status-bar", Static).update(text)

    @property
    def active_tab(self) -> str:
        return self.current_tab

    def action_next_tab(self) -> None:
        self._shift_tab(1)

    def action_previous_tab(self) -> None:
        self._shift_tab(-1)

    def _shift_tab(self, step: int) -> None:
        new_tab = self.TAB_ORDER[
            (self.TAB_ORDER.index(self.active_tab) + step) % len(self.TAB_ORDER)
        ]
        if new_tab == self.active_tab:
            return
        self.current_tab = new_tab
        if (
            self.active_tab == "history"
            and self.history_day is None
            and self.snapshot_data is not None
        ):
            self.history_day = self.snapshot_data.current_day
        if self.snapshot_data is not None and not self._pending_flash_messages:
            self.flash_message = self._status_summary()
        self._schedule_render()

    def action_refresh(self, announce: bool = True) -> None:
        if announce:
            self._set_flash("Refreshing snapshot")
        self._request_snapshot()

    def _selected_task(self) -> Task | None:
        rows_name = self.TASK_TABLES.get(self.active_tab)
        if rows_name is None:
            return None
        return self._selected_from_table(getattr(self, rows_name))

    def _selected_from_table(self, rows: list[Task]) -> Task | None:
        if not rows:
            return None
        row = self._selected_rows[self.active_tab]
        if row is None:
            return None
        return rows[row] if 0 <= row < len(rows) else None

    async def _run_mutation(self, backend_fn, *args, feedback_fn=None) -> None:
        async with self._mutation_lock:
            try:
                result = await asyncio.to_thread(backend_fn, *args)
            except BackendError as exc:
                self._show_feedback(str(exc), severity="error")
                return
            if feedback_fn:
                message = feedback_fn(result)
                self._pending_flash_messages.append(message)
                self.notify(message)
            await self._request_snapshot_future()

    def action_add_task(self) -> None:
        if self.active_tab != "today":
            return
        self.push_screen(TaskEditorScreen("Add Task"), callback=self._queue_add_task_result)

    def _queue_add_task_result(self, value: str | None) -> None:
        if value is None:
            return
        self.run_worker(
            self._run_mutation(self.backend.add, value, feedback_fn=lambda r: f"Added: {r.title}"),
            group="task-mutation",
            exit_on_error=False,
        )

    def action_edit_task(self) -> None:
        task = self._selected_task()
        if task is None:
            return
        self.push_screen(
            TaskEditorScreen("Edit Task", task.title),
            callback=lambda value: self._queue_edit_task_result(task, value),
        )

    def _queue_edit_task_result(self, task: Task, value: str | None) -> None:
        if value is None:
            return
        self.run_worker(
            self._run_mutation(
                self.backend.edit,
                task.id,
                value,
                feedback_fn=lambda _: f"Updated: {normalize_title(value)}",
            ),
            group="task-mutation",
            exit_on_error=False,
        )

    def action_delete_task(self) -> None:
        task = self._selected_task()
        if task is None:
            return
        self.push_screen(
            ConfirmScreen("Delete Task", task.title),
            callback=lambda confirmed: self._queue_delete_task_result(task, confirmed),
        )

    def _queue_delete_task_result(self, task: Task, confirmed: bool | None) -> None:
        if not confirmed:
            return
        self.run_worker(
            self._run_mutation(
                self.backend.delete, task.id, feedback_fn=lambda _: f"Deleted: {task.title}"
            ),
            group="task-mutation",
            exit_on_error=False,
        )

    async def action_done_task(self) -> None:
        task = self._selected_task()
        if task is None or self.active_tab != "today":
            return
        await self._run_mutation(
            self.backend.done, task.id, feedback_fn=lambda _: f"Completed: {task.title}"
        )

    async def action_abandon_task(self) -> None:
        task = self._selected_task()
        if task is None or self.active_tab != "today":
            return
        await self._run_mutation(
            self.backend.abandon, task.id, feedback_fn=lambda _: f"Abandoned: {task.title}"
        )

    async def action_postpone_task(self) -> None:
        task = self._selected_task()
        if task is None or self.active_tab != "today":
            return
        await self._run_mutation(
            self.backend.postpone,
            task.id,
            feedback_fn=lambda _: f"Postponed: {task.title} → tomorrow",
        )

    def action_history_prev_day(self) -> None:
        self._move_history_day(-1)

    def action_history_next_day(self) -> None:
        self._move_history_day(1)

    def _move_history_day(self, step: int) -> None:
        if self.active_tab != "history" or self.history_day is None:
            return
        selected_day = parse_day(self.history_day)
        new_day = selected_day + timedelta(days=step)
        if step < 0:
            if new_day < self.backend.MIN_HISTORY_DATE:
                return
            strip_start = parse_day(self.history_strip_start or self.history_day)
            self.history_day = new_day.isoformat()
            if selected_day <= strip_start:
                self.history_strip_start = self.history_day
            self.action_refresh(announce=False)
            return
        if self.snapshot_data is None:
            return
        current_day = parse_day(self.snapshot_data.current_day)
        if new_day > current_day:
            return
        strip_start = parse_day(self.history_strip_start or self.history_day)
        strip_end = strip_start + timedelta(days=self.HISTORY_WINDOW_DAYS - 1)
        self.history_day = new_day.isoformat()
        if selected_day < strip_end or strip_end >= current_day:
            self.action_refresh(announce=False)
            return
        self.history_strip_start = (strip_start + timedelta(days=1)).isoformat()
        self.action_refresh(announce=False)

    @on(DataTable.RowHighlighted)
    def _handle_row_highlighted(self, event: DataTable.RowHighlighted | None = None) -> None:
        if event is None:
            return
        if event.data_table.cursor_type == "none":
            return
        self._selected_rows[self.active_tab] = event.cursor_row

    def on_resize(self, _event: events.Resize) -> None:
        self._schedule_render()
