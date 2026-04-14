from .schema import Config, GeneralConfig, NotificationsConfig, KeybindingsConfig


def get_default_config() -> Config:
    return Config(
        general=GeneralConfig(),
        notifications=NotificationsConfig(),
        keybindings=KeybindingsConfig(),
        accounts=[],
        split_inboxes=[],
    )
