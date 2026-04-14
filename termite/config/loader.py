import tomllib
from pathlib import Path
import tomli_w
from pydantic import ValidationError
from .schema import Config

CONFIG_DIR = Path.home() / ".termite"
CONFIG_FILE = CONFIG_DIR / "config.toml"


def get_config_dir() -> Path:
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    return CONFIG_DIR


def load_config() -> Config:
    if not CONFIG_FILE.exists():
        return Config()

    try:
        with open(CONFIG_FILE, "rb") as f:
            data = tomllib.load(f)
        return Config(**data)
    except (tomllib.TOMLDecodeError, ValidationError) as e:
        print(f"Error loading config: {e}")
        return Config()


def save_config(config: Config) -> None:
    get_config_dir()
    data = config.model_dump(exclude_unset=True, exclude_defaults=False)
    with open(CONFIG_FILE, "wb") as f:
        tomli_w.dump(data, f)
