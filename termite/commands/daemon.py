from .registry import registry
from typing import Any
import sys
import platform
import os
from pathlib import Path


@registry.register(
    "daemon", "Manage the background sync daemon: /daemon install|status|stop"
)
async def daemon_command(args: str, app: Any) -> None:
    action = args.strip().lower()

    if action == "install":
        if platform.system() == "Darwin":
            _install_launchd()
            app.notify("Installed daemon to launchd.")
        elif platform.system() == "Linux":
            _install_systemd()
            app.notify("Installed daemon to systemd.")
        else:
            app.notify("Daemon install not supported on this OS.")
    elif action == "status":
        app.notify("Daemon status: check ~/.termite/status.json")
    elif action == "stop":
        if platform.system() == "Darwin":
            os.system(
                "launchctl unload -w ~/Library/LaunchAgents/com.termite.daemon.plist"
            )
        elif platform.system() == "Linux":
            os.system("systemctl --user stop termite.service")
        app.notify("Stopping daemon... (if running)")
    else:
        app.notify("Usage: /daemon install|status|stop")


def _install_launchd():
    plist = f"""<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.termite.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>{sys.executable}</string>
        <string>-m</string>
        <string>termite.cli</string>
        <string>daemon</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>"""
    launchagents = Path.home() / "Library" / "LaunchAgents"
    launchagents.mkdir(parents=True, exist_ok=True)
    with open(launchagents / "com.termite.daemon.plist", "w") as f:
        f.write(plist)
    os.system("launchctl load -w ~/Library/LaunchAgents/com.termite.daemon.plist")


def _install_systemd():
    service = f"""[Unit]
Description=Termite Background Sync Daemon

[Service]
ExecStart={sys.executable} -m termite.cli daemon
Restart=always
RestartSec=10

[Install]
WantedBy=default.target"""
    systemd_user = Path.home() / ".config" / "systemd" / "user"
    systemd_user.mkdir(parents=True, exist_ok=True)
    with open(systemd_user / "termite.service", "w") as f:
        f.write(service)
    os.system("systemctl --user daemon-reload")
    os.system("systemctl --user enable --now termite.service")
