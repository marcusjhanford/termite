from dataclasses import dataclass
from typing import Callable, Any


@dataclass
class Command:
    name: str
    description: str
    handler: Callable


class CommandRegistry:
    def __init__(self):
        self._commands: dict[str, Command] = {}

    def register(self, name: str, description: str):
        def decorator(fn: Callable):
            self._commands[name] = Command(name, description, fn)
            return fn

        return decorator

    async def dispatch(self, raw_input: str, app: Any) -> None:
        parts = raw_input.strip().lstrip("/").split(maxsplit=1)
        if not parts:
            return
        name = parts[0]
        args = parts[1] if len(parts) > 1 else ""
        if name in self._commands:
            await self._commands[name].handler(args, app)
        else:
            app.notify(f"Unknown command: {name}")


registry = CommandRegistry()
