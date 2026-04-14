from .registry import registry
from typing import Any
from ..config.schema import AccountConfig
from ..config.loader import save_config
from ..providers.gmail import GmailProvider
from ..providers.outlook import OutlookProvider
from ..engine.sync import SyncWorker


@registry.register(
    "connect", "Interactive wizard to add a new inbox: /connect <email> [gmail|outlook]"
)
async def connect_command(args: str, app: Any) -> None:
    parts = args.strip().split()
    if not parts:
        app.notify("Usage: /connect <email> [gmail|outlook]")
        return

    email = parts[0]
    provider_name = parts[1] if len(parts) > 1 else "gmail"

    if provider_name not in ["gmail", "outlook"]:
        app.notify("Provider must be 'gmail' or 'outlook'")
        return

    account_id = email.split("@")[0].lower()
    app.notify(f"Starting connection flow for {email} via {provider_name}...")

    # We use a worker so we don't block the UI thread during OAuth or Sync
    app.run_worker(_run_connect_flow(app, account_id, email, provider_name))


async def _run_connect_flow(
    app: Any, account_id: str, email: str, provider_name: str
) -> None:
    # 1. Setup account config
    account = AccountConfig(
        id=account_id, name=account_id.capitalize(), email=email, provider=provider_name
    )

    # 2. Run OAuth flow
    if provider_name == "gmail":
        provider = GmailProvider()
        try:
            app.notify("Running Gmail OAuth flow...")
            # Since this runs a local webserver, it blocks. The user has to click in browser.
            # Ensure client_secret.json is downloaded!
            await provider.run_auth_flow(account.id)
        except FileNotFoundError as e:
            app.notify(str(e))
            return
        except Exception as e:
            app.notify(f"Auth failed: {e}")
            return
    elif provider_name == "outlook":
        provider = OutlookProvider()
        try:
            app.notify("Starting Outlook Device flow...")
            await provider.run_auth_flow(account.id)
        except Exception as e:
            msg = str(e)
            if msg.startswith("DEVICE_CODE_FLOW:"):
                app.notify(msg.replace("DEVICE_CODE_FLOW:", ""), timeout=20)
                app.notify(
                    "Outlook auth flow requires device code (stubbed for MVP)",
                    timeout=10,
                )
                return
            else:
                app.notify(f"Auth failed: {e}")
                return

    # 3. Save to config
    app.config.accounts.append(account)
    save_config(app.config)
    app.notify(f"Account {email} added successfully!")

    # 4. Trigger sync
    app.notify(f"Starting initial sync for {email}...")
    sync_worker = SyncWorker()
    try:
        await sync_worker.initial_sync(account)
        app.notify("Initial sync completed!")

        # Refresh the UI if MainScreen is active
        # Assuming we just added the first account
        from ..ui.screens.main import MainScreen

        if isinstance(app.screen, MainScreen):
            app.run_worker(app.screen.load_threads("primary"))
    except Exception as e:
        app.notify(f"Sync failed: {e}")
