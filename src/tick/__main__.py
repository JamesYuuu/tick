from __future__ import annotations

import argparse

from .app import TickTextualApp
from .backend import TickBackend


def main() -> None:
    argparse.ArgumentParser(prog="tick").parse_args()
    app = TickTextualApp(TickBackend())
    app.run()


if __name__ == "__main__":
    main()
