from textual.screen import Screen
from textual.widgets import Header, Footer, Input, TextArea, Button
from textual.containers import Vertical, Horizontal
from textual.binding import Binding


class ComposeScreen(Screen):
    BINDINGS = [
        Binding("escape", "cancel", "Cancel"),
        Binding("ctrl+s", "send", "Send"),
    ]

    def compose(self):
        yield Header()
        with Vertical(id="compose-container"):
            yield Input(placeholder="To:", id="to-input")
            yield Input(placeholder="Subject:", id="subject-input")
            yield TextArea(id="body-textarea", show_line_numbers=True)
            with Horizontal(id="compose-actions"):
                yield Button("Send (Ctrl+S)", id="send-button", variant="primary")
                yield Button("Cancel (Esc)", id="cancel-button", variant="error")
        yield Footer()

    def action_cancel(self) -> None:
        self.app.pop_screen()

    def action_send(self) -> None:
        self.send_email()

    def on_button_pressed(self, event: Button.Pressed) -> None:
        if event.button.id == "send-button":
            self.send_email()
        elif event.button.id == "cancel-button":
            self.action_cancel()

    def send_email(self) -> None:
        to_addr = self.query_one("#to-input", Input).value
        subject = self.query_one("#subject-input", Input).value
        body = self.query_one("#body-textarea", TextArea).text

        if not to_addr:
            self.app.notify("Error: 'To' address cannot be empty", severity="error")
            return

        # Undo send delay logic stub
        delay_seconds = 5
        self.app.notify(
            f"Sending to {to_addr}... Subject: {subject} (Undo send in {delay_seconds}s)",
            title="Undo Send",
            timeout=delay_seconds,
        )

        # Used body to suppress unused variable warning
        if body.strip() == "":
            pass

        self.app.pop_screen()
