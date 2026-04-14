import json
import keyring
import msal
from .base import BaseProvider, Credentials

CLIENT_ID = "YOUR_CLIENT_ID"  # Placeholder for MSAL
AUTHORITY = "https://login.microsoftonline.com/common"
SCOPES = [
    "https://outlook.office.com/IMAP.AccessAsUser.All",
    "https://outlook.office.com/SMTP.Send",
]
KEYRING_SERVICE = "termite_outlook"


class OutlookProvider(BaseProvider):
    imap_host = "outlook.office365.com"
    imap_port = 993
    imap_ssl = True
    smtp_host = "smtp.office365.com"
    smtp_port = 587
    smtp_ssl = False

    def _save_token(self, account_id: str, result: dict) -> None:
        keyring.set_password(KEYRING_SERVICE, account_id, json.dumps(result))

    def _load_token(self, account_id: str) -> dict | None:
        token_data_str = keyring.get_password(KEYRING_SERVICE, account_id)
        if not token_data_str:
            return None
        return json.loads(token_data_str)

    def _build_msal_app(self) -> msal.PublicClientApplication:
        return msal.PublicClientApplication(CLIENT_ID, authority=AUTHORITY)

    async def get_credentials(self, account_id: str) -> Credentials:
        token_data = self._load_token(account_id)
        if not token_data or "access_token" not in token_data:
            raise ValueError(
                f"No credentials found for {account_id}. Run auth flow first."
            )

        # Refresh logic
        if "refresh_token" in token_data:
            app = self._build_msal_app()
            result = app.acquire_token_by_refresh_token(
                token_data["refresh_token"], scopes=SCOPES
            )
            if "access_token" in result:
                self._save_token(account_id, result)
                return Credentials(username="", oauth2_token=result["access_token"])

        return Credentials(username="", oauth2_token=token_data["access_token"])

    async def run_auth_flow(self, account_id: str) -> Credentials:
        app = self._build_msal_app()
        flow = app.initiate_device_flow(scopes=SCOPES)
        if "user_code" not in flow:
            raise ValueError("Failed to create device flow")

        # We need a way to pass this message back to the UI.
        # For now, we print it or throw an exception with the instructions.
        raise Exception(f"DEVICE_CODE_FLOW:{flow['message']}")

    async def continue_auth_flow(self, account_id: str, flow: dict) -> Credentials:
        app = self._build_msal_app()
        result = app.acquire_token_by_device_flow(flow)

        if "access_token" in result:
            self._save_token(account_id, result)
            return Credentials(username="", oauth2_token=result["access_token"])
        else:
            raise ValueError(f"Auth failed: {result.get('error_description')}")

    async def refresh_token(self, account_id: str) -> Credentials:
        return await self.get_credentials(account_id)
