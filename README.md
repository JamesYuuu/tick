# tick

Terminal todo app built with Textual and a native Python SQLite backend.

## Setup

```bash
python3 -m venv .venv
.venv/bin/pip install -e .[dev]
```

## Run

Start the TUI with the packaged CLI entrypoint:

```bash
.venv/bin/tick
```

Inspect the available CLI options:

```bash
PYTHONPATH=src .venv/bin/python -m tick --help
```

Use a custom database path when debugging or isolating a test run:

```bash
PYTHONPATH=src .venv/bin/python -m tick --db-path /tmp/tick-dev.db
```

Print the installed version without launching the UI:

```bash
PYTHONPATH=src .venv/bin/python -m tick --version
```

## Test

Run the backend regression suite required for Sprint 1:

```bash
PYTHONPATH=src .venv/bin/python -m pytest -q tests/test_backend.py
```

Run the full test suite:

```bash
PYTHONPATH=src .venv/bin/python -m pytest -q
```

## Data file location

By default Tick stores its SQLite database at:

- `$XDG_DATA_HOME/tick/todo.db` when `XDG_DATA_HOME` is set
- `~/.tick/todo.db` otherwise

Passing `--db-path` overrides the location completely, which is the recommended way to debug against a scratch database or to keep automated tests away from your real data.

## Debugging tips

- Use `--db-path` to reproduce issues with an isolated database file.
- Set `XDG_DATA_HOME` to a temporary directory when manually smoke-testing initialization behavior.
- If you need to inspect persisted data, open the SQLite file with `sqlite3 /path/to/todo.db`.
