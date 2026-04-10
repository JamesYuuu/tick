from __future__ import annotations

import asyncio
from typing import Any, Callable, Protocol

from .backend import BackendError, Snapshot


class DispatcherHost(Protocol):
    def run_worker(self, worker: Any, *, group: str, exit_on_error: bool = ...) -> Any: ...

    def load_snapshot(self) -> Snapshot: ...

    def apply_snapshot(self, snapshot: Snapshot) -> None: ...

    def show_feedback(self, message: str, *, severity: Any = ...) -> None: ...

    def notify(self, message: str, *, severity: Any = ...) -> None: ...

    def queue_flash_message(self, message: str) -> None: ...


class AppDispatcher:
    def __init__(self, host: DispatcherHost) -> None:
        self.host = host
        self.snapshot_loading = False
        self.snapshot_request_id = 0
        self.snapshot_processed_id = 0
        self.snapshot_waiters: dict[int, list[asyncio.Future[None]]] = {}
        self.mutation_lock = asyncio.Lock()

    def request_snapshot(self) -> None:
        self._enqueue_snapshot_request()

    def request_snapshot_future(self) -> asyncio.Future[None]:
        request_id = self._enqueue_snapshot_request()
        future: asyncio.Future[None] = asyncio.get_running_loop().create_future()
        self.snapshot_waiters.setdefault(request_id, []).append(future)
        return future

    async def run_mutation(
        self,
        backend_fn: Callable[..., Any],
        *args: object,
        feedback_fn: Callable[[Any], str] | None = None,
    ) -> None:
        async with self.mutation_lock:
            try:
                result = await asyncio.to_thread(backend_fn, *args)
            except BackendError as exc:
                self.host.show_feedback(str(exc), severity="error")
                return
            if feedback_fn:
                message = feedback_fn(result)
                self.host.queue_flash_message(message)
                self.host.notify(message)
            await self.request_snapshot_future()

    def _enqueue_snapshot_request(self) -> int:
        self.snapshot_request_id += 1
        self._start_snapshot_worker()
        return self.snapshot_request_id

    def _start_snapshot_worker(self) -> None:
        if self.snapshot_loading:
            return
        self.snapshot_loading = True
        self.host.run_worker(self._drain_snapshot_requests(), group="snapshot")

    async def _drain_snapshot_requests(self) -> None:
        try:
            while self.snapshot_processed_id < self.snapshot_request_id:
                target_request_id = self.snapshot_request_id
                try:
                    snapshot = await asyncio.to_thread(self.host.load_snapshot)
                except BackendError as exc:
                    self.host.show_feedback(str(exc), severity="error")
                    self.snapshot_processed_id = target_request_id
                    self._resolve_snapshot_waiters()
                else:
                    self.host.apply_snapshot(snapshot)
                    self.snapshot_processed_id = target_request_id
                    self._resolve_snapshot_waiters()
        finally:
            self.snapshot_loading = False
            if self.snapshot_processed_id < self.snapshot_request_id:
                self._start_snapshot_worker()

    def _resolve_snapshot_waiters(self) -> None:
        ready = [
            request_id
            for request_id in self.snapshot_waiters
            if request_id <= self.snapshot_processed_id
        ]
        for request_id in sorted(ready):
            for future in self.snapshot_waiters.pop(request_id):
                if not future.done():
                    future.set_result(None)
