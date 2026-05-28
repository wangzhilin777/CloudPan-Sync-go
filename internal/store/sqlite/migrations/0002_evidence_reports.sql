CREATE TABLE IF NOT EXISTS evidence_reports (
    id TEXT PRIMARY KEY,
    generated_at TEXT NOT NULL,
    title TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    markdown TEXT NOT NULL,
    summary_json TEXT NOT NULL,
    statuses_json TEXT NOT NULL,
    samples_json TEXT NOT NULL
);
