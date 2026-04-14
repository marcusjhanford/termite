from typing import Protocol
from dataclasses import dataclass


@dataclass
class Credentials:
    username: str
    password: str | None = None
    oauth2_token: str | None = None


class BaseProvider(Protocol):
    imap_host: str
    imap_port: int
    imap_ssl: bool
    smtp_host: str
    smtp_port: int
    smtp_ssl: bool

    async def get_credentials(self, account_id: str) -> Credentials: ...
    async def run_auth_flow(self, account_id: str) -> Credentials: ...
    async def refresh_token(self, account_id: str) -> Credentials: ...
