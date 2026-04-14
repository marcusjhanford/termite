import click
from .app import TermiteApp


@click.group(invoke_without_command=True)
@click.pass_context
def cli(ctx):
    """Termite - A keyboard-first TUI email client."""
    if ctx.invoked_subcommand is None:
        app = TermiteApp()
        app.run()


@cli.command()
def daemon():
    """Run the background sync daemon."""
    print("Running in daemon mode...")
    from termite.daemon import run_daemon
    import asyncio

    asyncio.run(run_daemon())


def main():
    cli()


if __name__ == "__main__":
    main()
