import json
from typing import Any
from ..config.loader import get_config_dir


async def write_status_file(cache: Any) -> None:
    status = {"unread": 0, "last_sync": "2023-10-14T09:32:00Z", "accounts": {}}
    status_file = get_config_dir() / "status.json"
    with open(status_file, "w") as f:
        json.dump(status, f)
