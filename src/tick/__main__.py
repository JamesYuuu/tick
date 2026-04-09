from __future__ import annotations

import argparse
from pathlib import Path

from . import __version__
from .app import TickTextualApp
from .backend import TickBackend


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="tick", description="Terminal todo app built with Textual."
    )
    parser.add_argument(
        "--db-path",
        type=Path,
        help="Override the SQLite database path (useful for tests and debugging).",
    )
    parser.add_argument(
        "--version",
        action="version",
        version=f"%(prog)s {__version__}",
    )
    return parser


def main() -> None:
    args = build_parser().parse_args()
    app = TickTextualApp(TickBackend(db_path=args.db_path))
    app.run()


if __name__ == "__main__":
    main()
