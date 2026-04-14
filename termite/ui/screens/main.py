from textual.screen import Screen
from textual.containers import Horizontal
from textual.widgets import Header, Footer, ListView, OptionList
from textual.binding import Binding
from ..widgets.inbox_list import InboxList
from ..widgets.thread_pane import ThreadPane, ThreadItem
from ..widgets.message_view import MessageView
from ..widgets.command_bar import CommandBar
from ...config import Config
from ...cache import get_db_pool, get_threads_for_inbox, get_messages_for_thread


class MainScreen(Screen):
    @classmethod
    def build_bindings(cls, config: Config) -> list[Binding]:
        kb = config.keybindings
        return [
            Binding(kb.compose, "compose", "Compose"),
            Binding(kb.reply, "reply", "Reply"),
            Binding(kb.archive, "archive", "Archive"),
            Binding(kb.delete, "delete", "Delete"),
            Binding(kb.next_thread, "next_thread", "Next"),
            Binding(kb.prev_thread, "prev_thread", "Prev"),
            Binding(kb.snooze, "snooze", "Snooze"),
            Binding(kb.search, "search", "Search"),
            Binding(kb.command, "command", "Command"),
            Binding(kb.inbox_zero, "inbox_zero", "Zero"),
        ]

    async def on_mount(self) -> None:
        self._bindings.bind_many(self.build_bindings(self.app.config))
        self.run_worker(self.load_threads("primary"))

    def compose(self):
        yield Header()
        with Horizontal():
            yield InboxList(id="inbox-list")
            yield ThreadPane(id="thread-pane")
            yield MessageView(id="message-view")
        yield CommandBar(id="command-bar")
        yield Footer()

    async def load_threads(self, split_inbox_id: str) -> None:
        db = await get_db_pool()
        threads = await get_threads_for_inbox(db, split_inbox_id)
        pane = self.query_one(ThreadPane)
        pane.clear_threads()

        # Inbox Zero Check
        if not threads:
            if getattr(self, "_seen_inbox_zero", False) is False:
                self._seen_inbox_zero = True
                from .inbox_zero import InboxZeroScreen

                self.app.push_screen(InboxZeroScreen())
            return

        self._seen_inbox_zero = False

        for t in threads:
            pane.add_thread(
                t["id"], t["subject"] or "No Subject", t["snippet"] or "No content"
            )

    async def on_list_view_selected(self, event: ListView.Selected) -> None:
        if isinstance(event.item, ThreadItem):
            await self.load_message(event.item.thread_id)

    async def load_message(self, thread_id: str) -> None:
        db = await get_db_pool()
        messages = await get_messages_for_thread(db, thread_id)
        view = self.query_one(MessageView)
        if messages:
            msg = messages[-1]  # Show the latest message in thread
            view.show_message(
                msg["subject"] or "No Subject",
                msg["body_text"] or msg["body_html"] or "",
            )

    def on_option_list_option_selected(self, event: OptionList.OptionSelected) -> None:
        # User selected an inbox from InboxList
        inbox_id = event.option.id
        if inbox_id:
            self.run_worker(self.load_threads(inbox_id))

    def action_command(self) -> None:
        cb = self.query_one(CommandBar)
        cb.display = True
        cb.focus()

    def action_search(self) -> None:
        cb = self.query_one(CommandBar)
        cb.display = True
        cb.value = "/search "
        cb.focus()

    def action_compose(self) -> None:
        from .compose import ComposeScreen

        self.app.push_screen(ComposeScreen())
