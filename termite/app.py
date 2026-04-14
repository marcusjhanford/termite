from textual.app import App
from .ui.screens.main import MainScreen
from .config.loader import load_config
from .commands.registry import registry
from .cache import init_db


class TermiteApp(App):
    CSS_PATH = "themes/dark.tcss"

    async def on_mount(self) -> None:
        await init_db()
        self.config = load_config()
        self.push_screen(MainScreen())

    async def dispatch_command(self, raw_input: str) -> None:
        await registry.dispatch(raw_input, self)


if __name__ == "__main__":
    app = TermiteApp()
    app.run()
