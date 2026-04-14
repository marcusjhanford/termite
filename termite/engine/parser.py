import mailparser
from dataclasses import dataclass


@dataclass
class ParsedMessage:
    message_id: str
    in_reply_to: str
    references: str
    subject: str
    date: int
    from_addr: str
    to_addrs: str
    cc_addrs: str
    body_text: str
    body_html: str
    raw_headers: str
    has_attachment: bool


def parse_raw_message(raw_bytes: bytes) -> ParsedMessage:
    mail = mailparser.parse_from_bytes(raw_bytes)

    # Extract headers
    headers = mail.headers
    message_id = headers.get("Message-ID", "")
    in_reply_to = headers.get("In-Reply-To", "")
    references = headers.get("References", "")
    subject = headers.get("Subject", "")

    # Dates
    date_timestamp = 0
    if mail.date:
        date_timestamp = int(mail.date.timestamp())

    # Addresses
    from_addr = ""
    if mail.from_:
        from_addr = mail.from_[0][1]  # Extract email address

    to_addrs = ",".join([addr[1] for addr in mail.to]) if mail.to else ""
    cc_addrs = ",".join([addr[1] for addr in mail.cc]) if mail.cc else ""

    # Bodies
    body_text = mail.text_plain[0] if mail.text_plain else ""
    body_html = mail.text_html[0] if mail.text_html else ""

    # Attachments
    has_attachment = len(mail.attachments) > 0

    # Raw headers (string representation)
    raw_headers_str = ""
    for k, v in headers.items():
        raw_headers_str += f"{k}: {v}\n"

    return ParsedMessage(
        message_id=message_id,
        in_reply_to=in_reply_to,
        references=references,
        subject=subject,
        date=date_timestamp,
        from_addr=from_addr,
        to_addrs=to_addrs,
        cc_addrs=cc_addrs,
        body_text=body_text,
        body_html=body_html,
        raw_headers=raw_headers_str,
        has_attachment=has_attachment,
    )
