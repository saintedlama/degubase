CREATE TABLE IF NOT EXISTS workspaces (
    id         INTEGER PRIMARY KEY,
    code       TEXT    NOT NULL DEFAULT '',
    name       TEXT    NOT NULL,
    context    TEXT    NOT NULL DEFAULT '',
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_workspaces_code ON workspaces(code);

CREATE TABLE IF NOT EXISTS tables (
    id              INTEGER PRIMARY KEY,
    workspace_id    INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    code            TEXT    NOT NULL DEFAULT '',
    name            TEXT    NOT NULL,
    context         TEXT    NOT NULL DEFAULT '',
    icon            TEXT    NOT NULL DEFAULT '',
    default_view_id INTEGER,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_tables_code ON tables(workspace_id, code);

CREATE TABLE IF NOT EXISTS columns (
    id       INTEGER PRIMARY KEY,
    table_id INTEGER NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    code     TEXT    NOT NULL DEFAULT '',
    name     TEXT    NOT NULL,
    type     TEXT    NOT NULL,
    options  TEXT,
    position INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_columns_slug ON columns(table_id, code);

CREATE TABLE IF NOT EXISTS rows (
    id         INTEGER PRIMARY KEY,
    table_id   INTEGER NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    data       TEXT    NOT NULL DEFAULT '{}',
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS views (
    id         INTEGER PRIMARY KEY,
    table_id   INTEGER NOT NULL REFERENCES tables(id) ON DELETE CASCADE,
    code       TEXT    NOT NULL DEFAULT '',
    name       TEXT    NOT NULL,
    type       TEXT    NOT NULL DEFAULT 'tabular',
    config     TEXT,
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS row_history (
    id          INTEGER PRIMARY KEY,
    row_id      INTEGER NOT NULL REFERENCES rows(id) ON DELETE CASCADE,
    data        TEXT    NOT NULL DEFAULT '{}',
    changed_at  TEXT    NOT NULL,
    revision_id TEXT,
    entry_type  TEXT    NOT NULL DEFAULT 'change',
    annotation  TEXT
);

CREATE INDEX IF NOT EXISTS idx_row_history_row_id      ON row_history(row_id);
CREATE INDEX IF NOT EXISTS idx_row_history_revision_id ON row_history(revision_id);

CREATE TABLE IF NOT EXISTS users (
    id         INTEGER PRIMARY KEY,
    username   TEXT    NOT NULL UNIQUE,
    name       TEXT    NOT NULL DEFAULT '',
    password   TEXT    NOT NULL,
    is_admin   INTEGER NOT NULL DEFAULT 0,
    created_at TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS workspace_tokens (
    id           INTEGER PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT    NOT NULL,
    token_hash   TEXT    NOT NULL UNIQUE,
    prefix       TEXT    NOT NULL,
    created_by   INTEGER REFERENCES users(id) ON DELETE SET NULL,
    created_at   TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workspace_tokens_ws ON workspace_tokens(workspace_id);

CREATE TABLE IF NOT EXISTS scripts (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT    NOT NULL,
    event_type   TEXT    NOT NULL,
    code         TEXT    NOT NULL DEFAULT '',
    enabled      INTEGER NOT NULL DEFAULT 1,
    created_at   TEXT    NOT NULL,
    updated_at   TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS script_tables (
    script_id INTEGER NOT NULL REFERENCES scripts(id) ON DELETE CASCADE,
    table_id  INTEGER NOT NULL REFERENCES tables(id)  ON DELETE CASCADE,
    PRIMARY KEY (script_id, table_id)
);

CREATE TABLE IF NOT EXISTS script_env_vars (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    key          TEXT    NOT NULL,
    value        TEXT    NOT NULL DEFAULT '',
    is_secret    INTEGER NOT NULL DEFAULT 0,
    UNIQUE(workspace_id, key)
);

CREATE TABLE IF NOT EXISTS script_executions (
    id                        INTEGER PRIMARY KEY AUTOINCREMENT,
    script_id                 INTEGER NOT NULL REFERENCES scripts(id) ON DELETE CASCADE,
    row_id                    INTEGER REFERENCES rows(id) ON DELETE SET NULL,
    event_type                TEXT    NOT NULL,
    started_at                TEXT    NOT NULL,
    ended_at                  TEXT    NOT NULL,
    duration_ms               INTEGER NOT NULL DEFAULT 0,
    success                   INTEGER NOT NULL DEFAULT 1,
    logs                      TEXT    NOT NULL DEFAULT '[]',
    depth                     INTEGER NOT NULL DEFAULT 0,
    triggered_by_execution_id INTEGER REFERENCES script_executions(id) ON DELETE SET NULL,
    table_code                TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS skills (
    id           INTEGER PRIMARY KEY,
    workspace_id INTEGER NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name         TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    table_ids    TEXT    NOT NULL DEFAULT '[]',
    operations   TEXT    NOT NULL DEFAULT '["read","create","update","delete"]',
    token_id     INTEGER REFERENCES workspace_tokens(id) ON DELETE SET NULL,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS row_links (
    source_row_id   INTEGER NOT NULL REFERENCES rows(id)    ON DELETE CASCADE,
    source_col_id   INTEGER NOT NULL REFERENCES columns(id) ON DELETE CASCADE,
    source_table_id INTEGER NOT NULL REFERENCES tables(id)  ON DELETE CASCADE,
    target_row_id   INTEGER NOT NULL,
    target_table_id INTEGER NOT NULL REFERENCES tables(id)  ON DELETE CASCADE,
    target_col_id   INTEGER REFERENCES columns(id) ON DELETE SET NULL,
    PRIMARY KEY (source_row_id, source_col_id)
);

CREATE INDEX IF NOT EXISTS idx_row_links_target ON row_links(target_row_id);

CREATE TABLE IF NOT EXISTS job_runs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    job_name    TEXT    NOT NULL,
    status      TEXT    NOT NULL DEFAULT 'running',
    outcome     TEXT    NOT NULL DEFAULT '',
    log         TEXT    NOT NULL DEFAULT '',
    started_at  TEXT    NOT NULL,
    finished_at TEXT
);
