from textual.screen import Screen
from textual.widgets import Static
from textual.binding import Binding
from rich.text import Text
from rich.style import Style


class InboxZeroScreen(Screen):
    BINDINGS = [
        Binding("escape", "dismiss", "Dismiss"),
        Binding("enter", "dismiss", "Dismiss"),
    ]

    def __init__(self, **kwargs):
        super().__init__(**kwargs)
        self.frame = 0
        # 16-bit sky colors (from light blue/yellow -> orange/pink -> dark purple/black)
        self.sky_colors = [
            "#87CEEB",
            "#87CEEB",
            "#87CEEB",
            "#87CEEB",  # Day
            "#FFB6C1",
            "#FFA07A",
            "#FF7F50",
            "#FF4500",  # Sunset
            "#8B008B",
            "#4B0082",
            "#483D8B",
            "#2F4F4F",  # Twilight
            "#191970",
            "#000080",
            "#000000",
            "#000000",  # Night
        ]
        self.sun_char = "██"
        self.sun_color = "#FFD700"  # Golden yellow

        self.forest_art = [
            "                                                 ",
            "       /\\                                        ",
            "      /  \\       /\\                /\\            ",
            "     /____\\     /  \\      /\\      /  \\           ",
            "    /      \\   /____\\    /  \\    /____\\    /\\    ",
            "   /________\\ /      \\  /____\\  /      \\  /  \\   ",
            "      ||     /________\\/      \\/________\\/____\\  ",
            "======||========||====/________\\===||======||====",
        ]
        self.forest_color = "#002200"  # Deep dark green

    def compose(self):
        yield Static(id="canvas", classes="inbox-zero-canvas")

    def on_mount(self) -> None:
        self.update_canvas()
        # ~10 FPS for a smooth but retro feel
        self.animation_timer = self.set_interval(0.1, self.tick)

    def tick(self) -> None:
        self.frame += 1
        self.update_canvas()

        # Stop animation when sun is fully set and night has fallen
        if self.frame >= len(self.sky_colors) * 2 + len(self.forest_art) + 10:
            self.animation_timer.pause()

    def update_canvas(self) -> None:
        canvas = self.query_one("#canvas", Static)
        width = self.app.console.size.width
        height = self.app.console.size.height

        if width < 50 or height < 20:
            canvas.update("Inbox Zero. See you soon.")
            return

        text = Text()

        # Calculate colors based on frame
        color_idx = min(self.frame // 2, len(self.sky_colors) - 1)
        sky_color = self.sky_colors[color_idx]

        # Sun position (descends over time)
        sun_y = 5 + (self.frame // 3)
        sun_x = width // 2 - 1

        # Draw sky and sun
        sky_height = height - len(self.forest_art) - 2  # leave room for ground/text
        for y in range(sky_height):
            row = ""
            for x in range(width):
                # Draw sun
                if y == sun_y and (x == sun_x or x == sun_x + 1):
                    pass  # handled below via spans
                # Draw stars if night
                elif color_idx >= len(self.sky_colors) - 4 and (x * y * 17) % 71 == 0:
                    row += "."
                elif color_idx >= len(self.sky_colors) - 2 and (x * y * 23) % 89 == 0:
                    row += "*"
                else:
                    row += " "

            # Sun logic
            if y == sun_y and sun_y < sky_height:
                prefix = row[:sun_x]
                suffix = row[sun_x + 2 :]
                line = Text(prefix, style=Style(bgcolor=sky_color, color="#ffffff"))
                line.append(
                    self.sun_char, style=Style(color=self.sun_color, bgcolor=sky_color)
                )
                line.append(
                    suffix + "\n", style=Style(bgcolor=sky_color, color="#ffffff")
                )
                text.append(line)
            else:
                text.append(row + "\n", style=Style(bgcolor=sky_color, color="#ffffff"))

        # Draw Forest
        forest_start_x = max(0, (width - 50) // 2)
        for i, line_art in enumerate(self.forest_art):
            padded_line = (
                " " * forest_start_x
                + line_art
                + " " * (width - forest_start_x - len(line_art))
            )
            # Pad to exact width
            padded_line = padded_line[:width]
            text.append(
                padded_line + "\n",
                style=Style(
                    color=self.forest_color,
                    bgcolor=sky_color
                    if i < len(self.forest_art) - 1
                    else self.forest_color,
                ),
            )

        # Ground / Text area
        text.append(" " * width + "\n", style=Style(bgcolor=self.forest_color))

        msg = "Inbox Zero achieved. The forest rests. See you soon."
        if self.frame > len(self.sky_colors) * 2:
            # Fade text in
            msg_padded = msg.center(width)
            text.append(
                msg_padded,
                style=Style(color="#ffffff", bgcolor=self.forest_color, bold=True),
            )
        else:
            text.append(" " * width, style=Style(bgcolor=self.forest_color))

        canvas.update(text)

    def action_dismiss(self) -> None:
        self.animation_timer.pause()
        self.app.pop_screen()
