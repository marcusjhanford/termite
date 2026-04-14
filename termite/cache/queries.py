import aiosqlite
from ..engine.parser import ParsedMessage


async def get_accounts(db: aiosqlite.Connection) -> list[aiosqlite.Row]:
    async with db.execute("SELECT * FROM accounts") as cursor:
        return await cursor.fetchall()


async def get_threads_for_inbox(
    db: aiosqlite.Connection, split_inbox_id: str, limit: int = 100
) -> list[aiosqlite.Row]:
    query = """
        SELECT * FROM threads
        WHERE split_inbox_id = ? AND is_deleted = 0 AND is_archived = 0
        ORDER BY last_message_at DESC
        LIMIT ?
    """
    async with db.execute(query, (split_inbox_id, limit)) as cursor:
        return await cursor.fetchall()


async def get_messages_for_thread(
    db: aiosqlite.Connection, thread_id: str
) -> list[aiosqlite.Row]:
    query = """
        SELECT * FROM messages
        WHERE thread_id = ?
        ORDER BY date ASC
    """
    async with db.execute(query, (thread_id,)) as cursor:
        return await cursor.fetchall()


async def search_messages(
    db: aiosqlite.Connection, query: str, limit: int = 50
) -> list[aiosqlite.Row]:
    sql = """
        SELECT messages.*, messages_fts.rank
        FROM messages_fts
        JOIN messages ON messages.rowid = messages_fts.rowid
        WHERE messages_fts MATCH ?
        ORDER BY rank
        LIMIT ?
    """
    async with db.execute(sql, (query, limit)) as cursor:
        return await cursor.fetchall()


async def save_thread_and_message(
    db: aiosqlite.Connection,
    account_id: str,
    folder: str,
    uid: int,
    parsed: ParsedMessage,
    thread_id: str,
    split_inbox_id: str = "primary",
) -> None:
    # Upsert thread
    snippet = parsed.body_text[:100] if parsed.body_text else parsed.body_html[:100]
    snippet = snippet.replace("\n", " ").strip()

    thread_sql = """
        INSERT INTO threads (id, account_id, subject, snippet, message_count, unread_count, has_attachment, last_message_at, split_inbox_id)
        VALUES (?, ?, ?, ?, 1, 1, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            message_count = message_count + 1,
            unread_count = unread_count + 1,
            last_message_at = excluded.last_message_at,
            snippet = excluded.snippet
    """
    await db.execute(
        thread_sql,
        (
            thread_id,
            account_id,
            parsed.subject,
            snippet,
            1 if parsed.has_attachment else 0,
            parsed.date,
            split_inbox_id,
        ),
    )

    # Insert message
    msg_id = parsed.message_id or f"{account_id}-{uid}"
    msg_sql = """
        INSERT OR IGNORE INTO messages (
            id, thread_id, account_id, uid, folder, from_addr, to_addrs, cc_addrs,
            subject, date, body_text, body_html, raw_headers, has_attachment, in_reply_to, references_
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    """
    await db.execute(
        msg_sql,
        (
            msg_id,
            thread_id,
            account_id,
            uid,
            folder,
            parsed.from_addr,
            parsed.to_addrs,
            parsed.cc_addrs,
            parsed.subject,
            parsed.date,
            parsed.body_text,
            parsed.body_html,
            parsed.raw_headers,
            1 if parsed.has_attachment else 0,
            parsed.in_reply_to,
            parsed.references,
        ),
    )
    await db.commit()
