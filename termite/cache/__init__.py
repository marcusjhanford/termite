from .db import get_db_pool, init_db
from .queries import (
    get_accounts,
    get_threads_for_inbox,
    get_messages_for_thread,
    search_messages,
    save_thread_and_message,
)

__all__ = [
    "get_db_pool",
    "init_db",
    "get_accounts",
    "get_threads_for_inbox",
    "get_messages_for_thread",
    "search_messages",
    "save_thread_and_message",
]
