from .registry import registry
from typing import Any
import json
from ..config.loader import get_config_dir


@registry.register(
    "sessions", "Manage named UI sessions: /sessions [list|save|restore] <name>"
)
async def sessions_command(args: str, app: Any) -> None:
    parts = args.strip().split()
    action = parts[0] if parts else "list"
    name = parts[1] if len(parts) > 1 else "default"

    sessions_file = get_config_dir() / "sessions.json"

    if action == "list":
        if sessions_file.exists():
            try:
                with open(sessions_file, "r") as f:
                    data = json.load(f)
                app.notify(f"Saved sessions: {', '.join(data.keys())}")
            except Exception:
                app.notify("No sessions found")
        else:
            app.notify("No sessions found")

    elif action == "save":
        data = {}
        if sessions_file.exists():
            try:
                with open(sessions_file, "r") as f:
                    data = json.load(f)
            except Exception:
                pass

        # Basic state save stub
        data[name] = {"inbox": "primary", "search_query": ""}
        with open(sessions_file, "w") as f:
            json.dump(data, f)
        app.notify(f"Session '{name}' saved")

    elif action == "restore":
        if sessions_file.exists():
            try:
                with open(sessions_file, "r") as f:
                    data = json.load(f)
                if name in data:
                    app.notify(f"Restored session '{name}'")
                else:
                    app.notify(f"Session '{name}' not found")
            except Exception:
                app.notify("Failed to restore session")
        else:
            app.notify("No sessions found")
    else:
        app.notify("Usage: /sessions [list|save|restore] <name>")
