import hashlib
from ..config.schema import AccountConfig
from ..cache.db import get_db_pool
from .imap import IMAPConnection
from ..providers.gmail import GmailProvider
from ..providers.outlook import OutlookProvider


class SyncWorker:
    def __init__(self):
        self.provider_map = {"gmail": GmailProvider(), "outlook": OutlookProvider()}

    def _get_provider(self, account: AccountConfig):
        return self.provider_map.get(account.provider)

    def _generate_thread_id(self, subject: str) -> str:
        # Simple subject-based fallback threading for now
        normalized_subject = (
            subject.lower().replace("re:", "").replace("fwd:", "").strip()
        )
        if not normalized_subject:
            normalized_subject = "no_subject"
        return hashlib.md5(normalized_subject.encode()).hexdigest()

    async def _sync_folder(
        self,
        account: AccountConfig,
        imap: IMAPConnection,
        folder: str,
        since_uid: int = 1,
    ) -> None:
        uids = await imap.fetch_uids_since(folder, since_uid)
        if not uids:
            return

        # For initial MVP, just fetch the last 10 if there are a lot, to speed up testing
        if len(uids) > 10:
            uids = uids[-10:]

        messages = await imap.fetch_messages(folder, uids)
        db = await get_db_pool()

        from ..cache.queries import save_thread_and_message

        for uid, parsed_msg in messages.items():
            thread_id = self._generate_thread_id(parsed_msg.subject)
            # Evaluate split inbox rules could happen here. Hardcoding 'primary' for now.
            split_inbox_id = "primary"

            await save_thread_and_message(
                db, account.id, folder, uid, parsed_msg, thread_id, split_inbox_id
            )

    async def initial_sync(self, account: AccountConfig) -> None:
        provider = self._get_provider(account)
        if not provider:
            print(f"Unknown provider: {account.provider}")
            return

        creds = await provider.get_credentials(account.id)

        imap = IMAPConnection()
        try:
            await imap.connect(
                provider.imap_host, provider.imap_port, provider.imap_ssl, creds
            )
            # Let's just sync the INBOX for the initial MVP
            await self._sync_folder(account, imap, "INBOX", 1)
        except Exception as e:
            print(f"Failed to sync {account.id}: {e}")
        finally:
            await imap.close()

    async def incremental_sync(self, account: AccountConfig) -> None:
        # In a real app, we'd query the DB for the max UID for the folder
        await self.initial_sync(account)

    async def idle_loop(self, account: AccountConfig) -> None:
        provider = self._get_provider(account)
        if not provider:
            return

        creds = await provider.get_credentials(account.id)
        imap = IMAPConnection()

        try:
            await imap.connect(
                provider.imap_host, provider.imap_port, provider.imap_ssl, creds
            )

            # Start IDLE mode on INBOX
            await imap.idle_start("INBOX")

            while True:
                # Wait for up to 29 minutes (RFC 2177 requires re-issuing IDLE at least every 29 mins)
                # But here we'll just check every 30 seconds for responsiveness
                responses = await imap.idle_check(timeout=30.0)

                if responses:
                    # e.g. [(12, b'EXISTS')] -> new message
                    has_new = any(b"EXISTS" in r for r in responses)
                    if has_new:
                        await imap.idle_done()
                        await self.incremental_sync(account)
                        await imap.idle_start("INBOX")

        except Exception as e:
            print(f"IDLE loop failed for {account.id}: {e}")
        finally:
            # We might need a separate try/except around idle_done if connection is dropped
            try:
                await imap.idle_done()
            except Exception:
                pass
            await imap.close()
