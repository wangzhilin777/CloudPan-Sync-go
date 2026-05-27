CREATE TABLE IF NOT EXISTS auth_profiles (
    id TEXT PRIMARY KEY,
    provider_key TEXT NOT NULL,
    auth_mode TEXT NOT NULL,
    display_name TEXT NOT NULL,
    token TEXT NOT NULL DEFAULT '',
    cookie TEXT NOT NULL DEFAULT '',
    extra_json TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'saved',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS auth_validations (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    provider_key TEXT NOT NULL,
    ok INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (profile_id) REFERENCES auth_profiles(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    source_provider TEXT NOT NULL,
    target_provider TEXT NOT NULL,
    state TEXT NOT NULL,
    completion_kind TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS task_items (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    strategy TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS task_results (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    task_item_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    mode TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS provider_probes (
    id TEXT PRIMARY KEY,
    provider_key TEXT NOT NULL,
    profile_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS provider_status_snapshots (
    id TEXT PRIMARY KEY,
    provider_key TEXT NOT NULL,
    summary_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
