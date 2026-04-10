from __future__ import annotations

from collections.abc import Sequence
from dataclasses import dataclass
from datetime import date


def parse_day(value: str) -> date:
    return date.fromisoformat(value)


def format_mmdd(value: str) -> str:
    return parse_day(value).strftime("%m-%d")


def format_top_bar_day(value: str) -> str:
    return parse_day(value).strftime("%Y-%m-%d %a")


def weighted_widths(total_width: int, weights: Sequence[int]) -> tuple[int, ...]:
    if total_width <= 0:
        return tuple(0 for _ in weights)
    total_weight = sum(weights)
    if total_weight <= 0:
        return tuple(0 for _ in weights)
    widths = [(total_width * weight) // total_weight for weight in weights]
    remainders = [(total_width * weight) % total_weight for weight in weights]
    remaining = total_width - sum(widths)
    for index in sorted(range(len(weights)), key=lambda i: remainders[i], reverse=True)[:remaining]:
        widths[index] += 1
    while any(width == 0 for width in widths) and any(width > 1 for width in widths):
        smallest = widths.index(0)
        largest = max(range(len(widths)), key=lambda i: widths[i])
        widths[largest] -= 1
        widths[smallest] += 1
    return tuple(widths)


@dataclass
class HistoryRow:
    marker: str
    title: str
    due_day: str
    context: str
