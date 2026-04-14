from typing import Any


def group_into_threads(
    messages: list[dict[str, Any]],
) -> dict[str, list[dict[str, Any]]]:
    """
    Stub for JWZ threading.
    For now, just groups by subject (ignoring Re:).
    """
    threads = {}
    for msg in messages:
        subject = msg.get("subject", "").replace("Re:", "").strip()
        if subject not in threads:
            threads[subject] = []
        threads[subject].append(msg)
    return threads
