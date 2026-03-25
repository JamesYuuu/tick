# tick

Terminal todo app built with Textual and a native Python SQLite backend.

## Setup

```bash
python3 -m venv .venv
.venv/bin/pip install -e .
```

## Run

```bash
.venv/bin/tick
```

## Test

```bash
PYTHONPATH=src .venv/bin/python -m unittest discover -s tests
```
