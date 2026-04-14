from textual.widgets import OptionList
from textual.widgets.option_list import Option


class InboxList(OptionList):
    def on_mount(self) -> None:
        self.add_options(
            [
                Option("Primary", id="primary"),
                Option("Notifications", id="notifications"),
                Option("Newsletters", id="newsletters"),
            ]
        )
