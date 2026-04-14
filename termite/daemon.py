import asyncio
from .config.loader import load_config
from .cache.db import get_db_pool
from .engine.sync import SyncWorker
from .notifications.status import write_status_file


async def run_daemon():
    config = load_config()
    db = await get_db_pool()
    sync_worker = SyncWorker()

    print("Daemon started. Syncing periodically...")
    while True:
        for account in config.accounts:
            try:
                # We do incremental sync
                await sync_worker.incremental_sync(account)
            except Exception as e:
                print(f"Daemon sync failed for {account.id}: {e}")

        try:
            # write status file to notify external tools
            await write_status_file(db)
        except Exception as e:
            print(f"Failed to write status file: {e}")

        await asyncio.sleep(config.general.check_interval_seconds)
