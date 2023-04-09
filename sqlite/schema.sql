CREATE TABLE IF NOT EXISTS members (
    id INTEGER PRIMARY KEY,
    slack_uid TEXT NOT NULL,
    received_likes INTEGER NOT NULL DEFAULT 0,
    received_dislikes INTEGER NOT NULL DEFAULT 0,
    received_reactions INTEGER NOT NULL DEFAULT 0,
    given_likes INTEGER NOT NULL DEFAULT 0,
    given_dislikes INTEGER NOT NULL DEFAULT 0,
    given_reactions INTEGER NOT NULL DEFAULT 0,
    month INTEGER NOT NULL,
    year INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(slack_uid, month, year)
);
