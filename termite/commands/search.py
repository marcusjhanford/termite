from .registry import registry
from typing import Any


@registry.register("search", "Full-text search across all cached messages")
async def search_command(args: str, app: Any) -> None:
    app.notify(f"Searching for: {args}")
