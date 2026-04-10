from __future__ import annotations

HELP_ROWS = {
    "today": (
        ("TASK", "x done", "b abandon", "p postpone"),
        ("EDIT", "a add", "e edit", "d delete"),
        ("FOCUS", "↑↓ move", "tab switch", "r refresh", "q quit"),
    ),
    "upcoming": (
        ("EDIT", "e edit", "d delete"),
        ("FOCUS", "↑↓ move", "tab switch", "r refresh", "q quit"),
    ),
    "history": (
        ("DATE", "← back", "→ forward"),
        ("FOCUS", "↑↓ move", "tab switch", "r refresh", "q quit"),
    ),
}


def render_help_text(active_tab: str, label_width: int, item_width: int) -> str:
    def fixed_row(label: str, *items: str) -> str:
        return (
            f"{label:<{label_width}}" + "".join(f"{item:<{item_width}}" for item in items).rstrip()
        )

    return "\n".join(fixed_row(*items) for items in HELP_ROWS[active_tab])
