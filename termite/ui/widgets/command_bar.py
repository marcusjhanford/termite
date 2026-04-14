from textual.widgets import Input
from textual.events import Key


class CommandBar(Input):
    def on_mount(self) -> None:
        self.display = False

    def on_key(self, event: Key) -> None:
        if event.key == "escape":
            self.display = False
            self.value = ""
            self.app.query_one("MainScreen").focus()

    async def on_input_submitted(self, message: Input.Submitted) -> None:
        self.display = False
        val = self.value
        self.value = ""
        self.app.query_one("MainScreen").focus()
        await self.app.dispatch_command(val)
