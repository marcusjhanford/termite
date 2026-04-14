from .registry import registry
from typing import Any


@registry.register(
    "snooze",
    "Snooze the selected thread (quick-pick: today / tomorrow / weekend / custom)",
)
async def snooze_command(args: str, app: Any) -> None:
    app.notify(f"Snoozing thread until {args or 'tomorrow'}... (stub)")
