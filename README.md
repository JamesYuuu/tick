# tuitodo

Terminal TUI todo app.

## Run

```bash
go run ./cmd/tuitodo
```

Data is stored in `~/.tuitodo/todo.db`.

## Views

- Today: active tasks due today or earlier (overdue tasks show in red)
- Upcoming: active tasks due after today
- History: done/abandoned by day + delayed ratios for the last 7 days

## Keys

- `1` Today, `2` Upcoming, `3` History
- `q` quit

Today:

- `a` add
- `x` done
- `d` abandon
- `p` postpone (+1 day)

History:

- `j/k` or `Down/Up` move day
- `h/l` or `Left/Right` shift 7-day window
