from pydantic import BaseModel, Field
from typing import Literal


class GeneralConfig(BaseModel):
    theme: str = "tokyo-night"
    editor: str = "vim"
    check_interval_seconds: int = 60
    startup_inbox: str = "primary"


class NotificationsConfig(BaseModel):
    desktop: bool = True
    terminal_bell: bool = False
    tmux_title: bool = True
    status_file: bool = True
    notify_on: Literal["unread", "all", "none"] = "unread"


class KeybindingsConfig(BaseModel):
    compose: str = "c"
    reply: str = "r"
    reply_all: str = "shift+r"
    forward: str = "f"
    archive: str = "e"
    delete: str = "#"
    mark_read: str = "m"
    mark_unread: str = "shift+m"
    snooze: str = "h"
    next_thread: str = "j"
    prev_thread: str = "k"
    open_thread: str = "enter"
    inbox_zero: str = "shift+i"
    search: str = "/"
    command: str = ":"
    quit: str = "q"


class AccountConfig(BaseModel):
    id: str
    name: str
    email: str
    provider: Literal["gmail", "outlook", "fastmail", "generic"]


class RuleConfig(BaseModel):
    field: Literal["from", "to", "subject", "list_unsubscribe", "header"]
    contains: list[str] | str | None = None
    not_contains: list[str] | str | None = None
    exists: bool | None = None
    header_name: str | None = None


class SplitInboxConfig(BaseModel):
    id: str
    label: str
    accounts: list[str]
    rules: list[RuleConfig]


class Config(BaseModel):
    general: GeneralConfig = Field(default_factory=GeneralConfig)
    notifications: NotificationsConfig = Field(default_factory=NotificationsConfig)
    keybindings: KeybindingsConfig = Field(default_factory=KeybindingsConfig)
    accounts: list[AccountConfig] = Field(default_factory=list)
    split_inboxes: list[SplitInboxConfig] = Field(default_factory=list)
