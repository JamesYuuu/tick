from __future__ import annotations

import os
import tempfile
from pathlib import Path

import pytest

from tick.backend import BackendError, TickBackend
from tick.history import weighted_widths

from conftest import FixedDayBackend


@pytest.fixture()
def backend(tmp_path: Path) -> FixedDayBackend:
    return FixedDayBackend(tmp_path / "todo.db", "2026-03-25")


def test_add_and_snapshot_places_task_in_today(backend: FixedDayBackend) -> None:
    task = backend.add("Write migration plan")
    snapshot = backend.snapshot()

    assert task.title == "Write migration plan"
    assert len(snapshot.today) == 1
    assert snapshot.today[0].title == "Write migration plan"
    assert snapshot.upcoming == []


def test_postpone_moves_task_to_upcoming(backend: FixedDayBackend) -> None:
    task = backend.add("Review schema")
    backend.postpone(task.id)

    snapshot = backend.snapshot()

    assert snapshot.today == []
    assert len(snapshot.upcoming) == 1
    assert snapshot.upcoming[0].due_day == "2026-03-26"


def test_done_updates_history_and_stats(backend: FixedDayBackend) -> None:
    task = backend.add("Close migration")
    backend.done(task.id)

    snapshot = backend.snapshot("2026-03-25")

    assert len(snapshot.history.done) == 1
    assert snapshot.history.done[0].title == "Close migration"
    assert snapshot.history.stats.total_done == 1
    assert snapshot.history.stats.delayed_done == 0


def test_overdue_active_task_stays_in_today_bucket(tmp_path: Path) -> None:
    db_path = tmp_path / "todo.db"
    creator = FixedDayBackend(db_path, "2026-03-25")
    creator.add("Review schema")

    snapshot = FixedDayBackend(db_path, "2026-03-26").snapshot()

    assert len(snapshot.today) == 1
    assert snapshot.today[0].title == "Review schema"
    assert snapshot.upcoming == []


def test_blank_title_is_rejected(backend: FixedDayBackend) -> None:
    with pytest.raises(BackendError):
        backend.add("   ")


def test_title_is_sanitized_before_storage(backend: FixedDayBackend) -> None:
    task = backend.add("  Write   migration   plan  ")

    snapshot = backend.snapshot()

    assert task.title == "Write migration plan"
    assert snapshot.today[0].title == "Write migration plan"


def test_edit_normalizes_title_before_storage(backend: FixedDayBackend) -> None:
    task = backend.add("Write migration plan")

    backend.edit(task.id, "  Review   migration   plan  ")
    snapshot = backend.snapshot()

    assert snapshot.today[0].title == "Review migration plan"


def test_invalid_operations_raise_backend_error(backend: FixedDayBackend) -> None:
    task = backend.add("Review schema")
    backend.done(task.id)

    with pytest.raises(BackendError, match="edit task"):
        backend.edit(task.id, "Renamed task")
    with pytest.raises(BackendError, match="delete task"):
        backend.delete(task.id)
    with pytest.raises(BackendError, match="postpone task"):
        backend.postpone(task.id)
    with pytest.raises(BackendError, match="mark abandoned"):
        backend.abandon(task.id)


def test_snapshot_rejects_invalid_history_day(backend: FixedDayBackend) -> None:
    with pytest.raises(BackendError, match="history_day must be a valid ISO date"):
        backend.snapshot("2026-02-30")


def test_snapshot_rejects_future_history_day(backend: FixedDayBackend) -> None:
    with pytest.raises(BackendError, match="history_day cannot be in the future"):
        backend.snapshot("2026-03-26")


def test_history_window_uses_selected_day_boundary(tmp_path: Path) -> None:
    backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-25")
    backend.add("Prepare launch notes")
    delayed_task = backend.add("Recover delayed task")
    backend.postpone(delayed_task.id)
    rollover_backend = FixedDayBackend(tmp_path / "todo.db", "2026-03-26")
    rollover_backend.done(delayed_task.id)

    snapshot = rollover_backend.snapshot("2026-03-26")

    assert snapshot.history.from_day == "2026-03-20"
    assert snapshot.history.to_day == "2026-03-26"
    assert snapshot.history.stats.total_done == 1
    assert snapshot.history.stats.delayed_done == 0


def test_history_stats_cover_done_abandoned_and_ratios(tmp_path: Path) -> None:
    db_path = tmp_path / "todo.db"
    day_20 = FixedDayBackend(db_path, "2026-03-20")
    done_on_time = day_20.add("Done on time")
    done_late = day_20.add("Done late")
    abandoned_on_time = day_20.add("Abandoned on time")
    abandoned_late = day_20.add("Abandoned late")
    day_20.done(done_on_time.id)
    day_20.abandon(abandoned_on_time.id)

    day_21 = FixedDayBackend(db_path, "2026-03-21")
    day_21.done(done_late.id)
    day_21.abandon(abandoned_late.id)

    stats = day_21.snapshot("2026-03-21").history.stats

    assert stats.total_done == 2
    assert stats.delayed_done == 1
    assert stats.total_abandoned == 2
    assert stats.delayed_abandoned == 1
    assert stats.done_delayed_ratio == 0.5
    assert stats.abandoned_delayed_ratio == 0.5


def test_repeated_postpone_keeps_advancing_due_day_stably(backend: FixedDayBackend) -> None:
    task = backend.add("Review schema")

    backend.postpone(task.id)
    backend.postpone(task.id)
    backend.postpone(task.id)

    snapshot = backend.snapshot()

    assert snapshot.today == []
    assert len(snapshot.upcoming) == 1
    assert snapshot.upcoming[0].due_day == "2026-03-28"


def test_default_db_path_prefers_xdg_data_home() -> None:
    with tempfile.TemporaryDirectory() as temp_dir:
        original = os.environ.get("XDG_DATA_HOME")
        os.environ["XDG_DATA_HOME"] = temp_dir
        try:
            backend = TickBackend()
        finally:
            if original is None:
                os.environ.pop("XDG_DATA_HOME", None)
            else:
                os.environ["XDG_DATA_HOME"] = original

        assert str(backend.db_path).startswith(temp_dir)
        assert backend.db_path == Path(temp_dir) / "tick" / "todo.db"


def test_weighted_widths_with_zero_total_width() -> None:
    assert weighted_widths(0, (1, 1)) == (0, 0)
    assert weighted_widths(-5, (1, 4, 2)) == (0, 0, 0)


def test_weighted_widths_with_zero_weights() -> None:
    assert weighted_widths(100, (0, 0, 0)) == (0, 0, 0)
    assert weighted_widths(0, (0, 0)) == (0, 0)
