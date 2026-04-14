from pathlib import Path
from textual.app import App


class ThemeManager:
    def __init__(self):
        self.themes_dir = Path(__file__).parent.parent / "themes"

    def discover(self) -> list[str]:
        if not self.themes_dir.exists():
            return []
        return [f.stem for f in self.themes_dir.glob("*.tcss")]

    def apply(self, name: str, app: App) -> None:
        theme_path = self.themes_dir / f"{name}.tcss"
        if theme_path.exists():
            try:
                # Based on Textual's standard styling
                app.stylesheet.read(str(theme_path))
                app.notify(f"Theme switched to {name}")
            except Exception as e:
                app.notify(f"Failed to apply theme: {e}")
        else:
            app.notify(f"Theme {name} not found.")


theme_manager = ThemeManager()
