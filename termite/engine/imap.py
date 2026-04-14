import asyncio
from imapclient import IMAPClient
from ..providers.base import Credentials
from .parser import ParsedMessage, parse_raw_message


class IMAPConnection:
    def __init__(self):
        self.client: IMAPClient | None = None
        self.loop = asyncio.get_running_loop()

    async def connect(
        self, host: str, port: int, ssl: bool, credentials: Credentials
    ) -> None:
        def _connect():
            client = IMAPClient(host, port=port, ssl=ssl)
            if credentials.oauth2_token:
                # IMAPClient expects oauth2_login(user, token)
                client.oauth2_login(credentials.username, credentials.oauth2_token)
            else:
                client.login(credentials.username, credentials.password)
            return client

        self.client = await self.loop.run_in_executor(None, _connect)

    async def list_folders(self) -> list[str]:
        def _list():
            return [folder[2] for folder in self.client.list_folders()]

        return await self.loop.run_in_executor(None, _list)

    async def fetch_uids_since(self, folder: str, since_uid: int) -> list[int]:
        def _fetch():
            self.client.select_folder(folder, readonly=True)
            return self.client.search([f"UID {since_uid}:*"])

        return await self.loop.run_in_executor(None, _fetch)

    async def fetch_messages(
        self, folder: str, uids: list[int]
    ) -> dict[int, ParsedMessage]:
        if not uids:
            return {}

        def _fetch():
            self.client.select_folder(folder, readonly=True)
            # fetch raw message
            raw_data = self.client.fetch(uids, ["RFC822", "FLAGS"])
            parsed = {}
            for uid, msg_data in raw_data.items():
                if b"RFC822" in msg_data:
                    parsed[uid] = parse_raw_message(msg_data[b"RFC822"])
            return parsed

        return await self.loop.run_in_executor(None, _fetch)

    async def idle_start(self, folder: str) -> None:
        def _idle_start():
            self.client.select_folder(folder, readonly=True)
            self.client.idle()

        await self.loop.run_in_executor(None, _idle_start)

    async def idle_check(self, timeout: float = 29.0) -> list[tuple]:
        def _idle_check():
            return self.client.idle_check(timeout=timeout)

        return await self.loop.run_in_executor(None, _idle_check)

    async def idle_done(self) -> None:
        def _idle_done():
            self.client.idle_done()

        await self.loop.run_in_executor(None, _idle_done)

    async def close(self) -> None:
        if self.client:
            await self.loop.run_in_executor(None, self.client.logout)
