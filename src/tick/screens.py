from __future__ import annotations

from textual.app import ComposeResult
from textual.containers import Container
from textual.screen import ModalScreen
from textual.widgets import Input, Label, Static


class TaskEditorScreen(ModalScreen[str | None]):
    def __init__(self, title: str, initial_value: str = "") -> None:
        super().__init__()
        self.screen_title = title
        self.initial_value = initial_value

    def compose(self) -> ComposeResult:
        with Container(id="modal-shell"):
            yield Label(self.screen_title, id="modal-title")
            yield Input(value=self.initial_value, placeholder="Task title", id="task-input")
            yield Label("Enter submit, Esc cancel", id="modal-help")

    def on_mount(self) -> None:
        self.query_one(Input).focus()

    def on_input_submitted(self, event: Input.Submitted) -> None:
        self.dismiss(event.value)

    def key_escape(self) -> None:
        self.dismiss(None)


class ConfirmScreen(ModalScreen[bool]):
    def __init__(self, title: str, body: str) -> None:
        super().__init__()
        self.screen_title = title
        self.body = body

    def compose(self) -> ComposeResult:
        with Container(id="modal-shell", classes="confirm"):
            yield Label(self.screen_title, id="modal-title")
            yield Static(self.body, id="confirm-body")
            yield Label("y confirm, n / Esc cancel", id="modal-help")

    def key_y(self) -> None:
        self.dismiss(True)

    def key_n(self) -> None:
        self.dismiss(False)

    def key_escape(self) -> None:
        self.dismiss(False)
