from __future__ import annotations

from collections.abc import Sequence

from textual.widgets import DataTable

from .backend import Snapshot, Task
from .history import HistoryRow, format_mmdd, parse_day, weighted_widths

HISTORY_STATE_COLUMN_WIDTH = 8
TABLE_ROW_HEIGHT = 3
TABLE_HEADER_HEIGHT = 3


def padded_text(value: str) -> str:
    return f"\n{value}\n"


def column_specs(active_tab: str) -> tuple[tuple[str, str, int | None], ...]:
    if active_tab == "history":
        return (
            ("State", "state", HISTORY_STATE_COLUMN_WIDTH),
            ("Task", "task", None),
            ("Due", "due", None),
        )
    return (("Task", "task", None), ("Due", "due", None))


def task_due_label(task: Task, current_day: str | None) -> str:
    if current_day is None:
        return format_mmdd(task.due_day)
    delta = (parse_day(task.due_day) - parse_day(current_day)).days
    if delta < 0:
        return f"{format_mmdd(task.due_day)}  overdue by {abs(delta)}d"
    if delta == 0:
        return f"{format_mmdd(task.due_day)}  due today"
    return f"{format_mmdd(task.due_day)}  in {delta}d"


def task_cells(task: Task, current_day: str | None) -> tuple[str, str]:
    return (padded_text(task.title), padded_text(task_due_label(task, current_day)))


def history_cells(row: HistoryRow) -> tuple[str, str, str]:
    due = f"{format_mmdd(row.due_day)}  {row.context}"
    return (
        padded_text(row.marker),
        padded_text(row.title),
        padded_text(due),
    )


def render_active_row(row: object, current_day: str | None) -> tuple[str, ...]:
    if isinstance(row, HistoryRow):
        return history_cells(row)
    if not isinstance(row, Task):
        raise TypeError(f"unsupported row type: {type(row)!r}")
    return task_cells(row, current_day)


def empty_row(active_tab: str) -> tuple[str, ...]:
    if active_tab == "today":
        return (
            padded_text("Nothing due today."),
            padded_text("Press a in Today to add one."),
        )
    if active_tab == "upcoming":
        return (
            padded_text("No upcoming tasks."),
            padded_text("Press a in Today to add one."),
        )
    return (
        padded_text("·"),
        padded_text("No history updates for this day."),
        padded_text("Try ← / → to browse."),
    )


def build_history_rows(snapshot: Snapshot) -> list[HistoryRow]:
    rows: list[HistoryRow] = []
    for t in snapshot.history.done:
        rows.append(HistoryRow("done", t.title, t.due_day, "finished"))
    for t in snapshot.history.abandoned:
        rows.append(HistoryRow("drop", t.title, t.due_day, "dropped"))
    for t in snapshot.history.active_due:
        rows.append(HistoryRow("late", t.title, t.due_day, "still open"))
    return rows


def balance_columns(table: DataTable, active_tab: str) -> None:
    if active_tab == "history":
        _balance_history_columns(table)
    else:
        _balance_task_due_columns(table)


def _balance_task_due_columns(table: DataTable) -> None:
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


def _balance_history_columns(table: DataTable) -> None:
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


class TableView:
    def __init__(self, table: DataTable) -> None:
        self._table = table
        self._configured_tab: str | None = None

    def prepare(self) -> None:
        self._table.show_horizontal_scrollbar = False

    @property
    def configured_tab(self) -> str | None:
        return self._configured_tab

    def configure(self, active_tab: str) -> bool:
        if self._configured_tab == active_tab:
            return False
        self._table.clear(columns=True)
        for label, key, width in column_specs(active_tab):
            if width is None:
                self._table.add_column(padded_text(label), key=key)
            else:
                self._table.add_column(padded_text(label), key=key, width=width)
        self._configured_tab = active_tab
        return True

    def fill(
        self,
        active_tab: str,
        rows: Sequence[object],
        current_day: str | None,
        selected_row: int | None,
    ) -> int | None:
        previous = selected_row if selected_row is not None else 0
        self._table.clear(columns=False)
        if not rows:
            self._table.add_row(*empty_row(active_tab), height=TABLE_ROW_HEIGHT)
            self._table.cursor_type = "none"
            return None
        self._table.cursor_type = "row"
        active_row = min(previous, len(rows) - 1)
        for row in rows:
            self._table.add_row(*render_active_row(row, current_day), height=TABLE_ROW_HEIGHT)
        self._table.move_cursor(row=active_row, animate=False)
        return active_row

    def balance_columns(self, active_tab: str) -> None:
        balance_columns(self._table, active_tab)
