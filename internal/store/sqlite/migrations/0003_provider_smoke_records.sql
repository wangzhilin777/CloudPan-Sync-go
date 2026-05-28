CREATE TABLE IF NOT EXISTS provider_smoke_records (
    id TEXT PRIMARY KEY,
    provider_key TEXT NOT NULL,
    protocol_group TEXT NOT NULL DEFAULT '',
    auth_mode TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL,
    title TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    operations_json TEXT NOT NULL,
    environment_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);
