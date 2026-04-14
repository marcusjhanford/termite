import aiosqlite
from pathlib import Path
from ..config.loader import get_config_dir

DB_PATH = get_config_dir() / "cache.db"
SCHEMA_FILE = Path(__file__).parent / "schema.sql"

_db_connection = None


async def get_db_pool() -> aiosqlite.Connection:
    global _db_connection
    if _db_connection is None:
        _db_connection = await aiosqlite.connect(DB_PATH)
        _db_connection.row_factory = aiosqlite.Row
    return _db_connection


async def init_db() -> None:
    db = await get_db_pool()
    async with db.execute(
        "SELECT name FROM sqlite_master WHERE type='table' AND name='accounts'"
    ) as cursor:
        if not await cursor.fetchone():
            with open(SCHEMA_FILE, "r") as f:
                schema_sql = f.read()
            await db.executescript(schema_sql)
            await db.commit()
