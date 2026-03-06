# TUI Todo Design (Daily, Delayed Highlight)

## Goal

Build a terminal-based (TUI) todo app that tracks tasks by day, supports three outcomes for each task (done, not done/active, abandoned), and highlights delayed tasks in red.

Key product rules:

- A task that is marked done disappears from the active list.
- A task that is abandoned disappears from the active list, but remains available in history and statistics.
- "Advance Day" is driven by system time (local date), not by a user command.
- A task is considered delayed if `current_day > due_day`.
- Creating a task defaults `due_day = current_day`.
- If the user actively postpones a task, `due_day` moves forward (so it does not become delayed due to that postponement).

## Non-Goals (v1)

- Multi-device sync
- Multi-user
- Natural language parsing
- Recurring tasks/habits (can be layered on later)

## Data Model

### Task

Fields (conceptual):

- `id`: stable identifier
- `title`: string
- `status`: enum `active | done | abandoned`
- `created_day`: local date `YYYY-MM-DD`
- `due_day`: local date `YYYY-MM-DD`
- `done_day`: local date `YYYY-MM-DD` (nullable)
- `abandoned_day`: local date `YYYY-MM-DD` (nullable)

Derived flags:

- `is_delayed(task, current_day) = (task.status == active) && (task.due_day < current_day)`

### Time Basis

- `current_day` is computed from system time using a configured timezone (default: local system timezone).
- The UI periodically checks for date rollover; on rollover it refreshes views and re-evaluates delayed status.

## Core Views

### Today (default)

Query:

- Show tasks where `status == active && due_day <= current_day`.

Rendering:

- Tasks with `due_day == current_day`: normal styling.
- Tasks with `due_day < current_day`: red/highlighted styling (delayed).

Actions:

- Add task: create with `due_day = current_day`.
- Done: set `status=done`, set `done_day=current_day`; remove from Today view.
- Abandon: set `status=abandoned`, set `abandoned_day=current_day`; remove from Today view.
- Postpone: move `due_day` forward (default +1 day).

### Upcoming (hidden by default)

Query:

- Show tasks where `status == active && due_day > current_day`.

Purpose:

- Visibility into intentionally scheduled tasks without cluttering Today.

### History

Provide day-based review and stats:

- For a selected day D, list tasks completed (`done_day == D`) and abandoned (`abandoned_day == D`).
- Provide totals per day: done count, abandoned count.

Delayed outcome stats (required):

- A delayed outcome is when a task is done/abandoned on day D and `due_day < D`.
- For any chosen range (e.g., day D, week, month), compute:
  - done_delayed_ratio = delayed_done / total_done
  - abandoned_delayed_ratio = delayed_abandoned / total_abandoned

## Behavioral Rules

### "Done disappears"

- The active lists are defined purely by `status == active`, so transitions to `done` or `abandoned` remove tasks from active views.

### Postpone (user action)

- Default behavior: `due_day = due_day + 1 day` for the selected task.
- Optional enhancements: postpone by 7 days, or set an explicit date.

### System day rollover

- Triggered by system time crossing midnight in the configured timezone.
- No bulk mutation of tasks is required.
- The change in `current_day` naturally causes:
  - tasks with `due_day < current_day` to become delayed and appear in Today.
  - tasks with `due_day == current_day` to appear in Today.

## Storage

Persist tasks locally in a single file-based datastore (details to be chosen in implementation plan):

- Must support querying by `status`, `due_day`, and by outcome days (`done_day`, `abandoned_day`).
- Must be crash-safe (atomic write or journaling).

## UX Notes (TUI)

- Use clear color semantics:
  - Normal tasks: default foreground.
  - Delayed tasks: red/high-contrast.
  - Done/abandoned are not shown in Today/Upcoming; only in History.
- Provide quick view switching: Today <-> Upcoming <-> History.

## Open Questions (deferred)

- Task ordering (created order vs. due_day then created)
- Search/filter within views
- Editing titles
- Bulk operations
