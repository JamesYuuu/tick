# tick

Terminal TUI todo app.

## Run

```bash
go run ./cmd/tick
```

Data is stored in `~/.tick/todo.db`.

## Views

- Today: active tasks due today or earlier (overdue tasks show in red)
- Upcoming: active tasks due after today
- History: done/abandoned by day + delayed ratios for the last 7 days

## Keys

- `tab` next view (Today -> Upcoming -> History)
- `q` quit

Today:

- `a` add
- `x` done
- `d` abandon
- `p` postpone (+1 day)

History:

- `h/l` or `Left/Right` move day
- `j/k` or `Down/Up` scroll tasks
