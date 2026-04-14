import json
import keyring
from pathlib import Path
from google_auth_oauthlib.flow import InstalledAppFlow
from google.oauth2.credentials import Credentials as GoogleCredentials
from google.auth.transport.requests import Request
from .base import BaseProvider, Credentials
from ..config.loader import get_config_dir

SCOPES = ["https://mail.google.com/"]
KEYRING_SERVICE = "termite_gmail"


class GmailProvider(BaseProvider):
    imap_host = "imap.gmail.com"
    imap_port = 993
    imap_ssl = True
    smtp_host = "smtp.gmail.com"
    smtp_port = 587
    smtp_ssl = False

    def _get_client_secrets_path(self) -> Path:
        return get_config_dir() / "client_secret.json"

    def _save_token(self, account_id: str, creds: GoogleCredentials) -> None:
        token_data = {
            "token": creds.token,
            "refresh_token": creds.refresh_token,
            "token_uri": creds.token_uri,
            "client_id": creds.client_id,
            "client_secret": creds.client_secret,
            "scopes": creds.scopes,
        }
        keyring.set_password(KEYRING_SERVICE, account_id, json.dumps(token_data))

    def _load_token(self, account_id: str) -> GoogleCredentials | None:
        token_data_str = keyring.get_password(KEYRING_SERVICE, account_id)
        if not token_data_str:
            return None

        token_data = json.loads(token_data_str)
        return GoogleCredentials(
            token=token_data["token"],
            refresh_token=token_data["refresh_token"],
            token_uri=token_data["token_uri"],
            client_id=token_data["client_id"],
            client_secret=token_data["client_secret"],
            scopes=token_data["scopes"],
        )

    async def get_credentials(self, account_id: str) -> Credentials:
        google_creds = self._load_token(account_id)
        if not google_creds:
            raise ValueError(
                f"No credentials found for {account_id}. Run auth flow first."
            )

        if google_creds.expired and google_creds.refresh_token:
            google_creds.refresh(Request())
            self._save_token(account_id, google_creds)

        return Credentials(username="unknown", oauth2_token=google_creds.token)

    async def run_auth_flow(self, account_id: str) -> Credentials:
        secrets_path = self._get_client_secrets_path()
        if not secrets_path.exists():
            raise FileNotFoundError(
                f"Missing {secrets_path}. Please download your OAuth 2.0 Client ID JSON from Google Cloud Console."
            )

        flow = InstalledAppFlow.from_client_secrets_file(str(secrets_path), SCOPES)
        google_creds = flow.run_local_server(port=8765)

        self._save_token(account_id, google_creds)
        # Note: We need the email address. We could fetch it using the token,
        # but for now we assume it's set in the account config.
        return Credentials(username="", oauth2_token=google_creds.token)

    async def refresh_token(self, account_id: str) -> Credentials:
        return await self.get_credentials(account_id)
