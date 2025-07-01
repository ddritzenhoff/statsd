CREATE TABLE IF NOT EXISTS members (
    id INTEGER PRIMARY KEY,
    month_year TEXT NOT NULL,
    slack_uid TEXT NOT NULL,
    received_likes INTEGER NOT NULL DEFAULT 0,
    received_dislikes INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(slack_uid, month_year)
);
