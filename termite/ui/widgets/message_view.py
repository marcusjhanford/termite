from textual.widgets import Static
from textual.containers import VerticalScroll


class MessageView(VerticalScroll):
    def on_mount(self) -> None:
        self.mount(Static("Select a thread to view the message.", id="message-body"))

    def show_message(self, subject: str, body: str) -> None:
        body_widget = self.query_one("#message-body", Static)
        content = f"[b]{subject}[/b]\n\n{body}"
        body_widget.update(content)
