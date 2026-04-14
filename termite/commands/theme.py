from .registry import registry
from typing import Any
from ..ui.theme_manager import theme_manager


@registry.register("theme", "Switch, list, or edit themes")
async def theme_command(args: str, app: Any) -> None:
    if args.strip() == "list":
        themes = theme_manager.discover()
        app.notify(f"Themes: {', '.join(themes)}")
    elif args.strip() == "edit":
        app.notify("Opening theme in editor...")
    elif args.strip():
        theme_manager.apply(args.strip(), app)
    else:
        app.notify("Usage: :theme <name|list|edit>")
