from textual.widgets import ListView, ListItem, Label
from textual.containers import Vertical


class ThreadItem(ListItem):
    def __init__(self, thread_id: str, subject: str, sender: str, **kwargs):
        super().__init__(**kwargs)
        self.thread_id = thread_id
        self.subject = subject
        self.sender = sender

    def compose(self):
        yield Vertical(
            Label(self.sender, classes="thread-sender"),
            Label(self.subject, classes="thread-subject"),
        )


class ThreadPane(ListView):
    def clear_threads(self) -> None:
        self.clear()

    def add_thread(self, thread_id: str, subject: str, snippet: str) -> None:
        # We'll use the snippet as sender/preview for MVP
        self.append(ThreadItem(thread_id, subject, snippet))
