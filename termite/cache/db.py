import aiosqlite
from pathlib import Path
from ..config.loader import get_config_dir

DB_PATH = get_config_dir() / "cache.db"
SCHEMA_FILE = Path(__file__).parent / "schema.sql"


async def get_db_pool() -> aiosqlite.Connection:
    db = await aiosqlite.connect(DB_PATH)
    db.row_factory = aiosqlite.Row
    return db


async def init_db() -> None:
    async with await get_db_pool() as db:
        async with db.execute(
            "SELECT name FROM sqlite_master WHERE type='table' AND name='accounts'"
        ) as cursor:
            if not await cursor.fetchone():
                with open(SCHEMA_FILE, "r") as f:
                    schema_sql = f.read()
                await db.executescript(schema_sql)
                await db.commit()
