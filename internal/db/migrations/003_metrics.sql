CREATE TABLE IF NOT EXISTS daily_metrics (
    date           TEXT,
    account_id     TEXT,
    emails_cleared INTEGER DEFAULT 0,
    emails_sent    INTEGER DEFAULT 0,
    inbox_zeros    INTEGER DEFAULT 0,
    time_in_app_s  INTEGER DEFAULT 0,
    streak_days    INTEGER DEFAULT 0,
    PRIMARY KEY (date, account_id)
);

CREATE TABLE IF NOT EXISTS milestones (
    id             TEXT PRIMARY KEY,
    unlocked_at    INTEGER,
    shown          INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS milestone_definitions (
    id             TEXT PRIMARY KEY,
    category       TEXT,
    threshold      INTEGER,
    label          TEXT,
    description    TEXT,
    icon           TEXT
);
