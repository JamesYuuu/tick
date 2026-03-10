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
- History: done/abandoned by day, with delayed ratios shown in the History body for the last 7 days

The footer uses two grouped help lines in every view: view/task actions first, global navigation second, with emphasized key tokens and lowercase labels.

## Keys

- `tab` next view (Today -> Upcoming -> History)
- `q` quit

Today:

- `a` add
- `e` edit selected task title
- `delete` or `backspace` delete selected task
- `x` done
- `d` abandon
- `p` postpone

Upcoming:

- `e` edit selected task title
- `delete` or `backspace` delete selected task

History:

- `h/l` or `left/right` prev day/next day
- `j/k` or `down/up` scroll tasks
