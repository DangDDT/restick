-- Initial single-operator factory schema (spec §8.2).
-- Timestamps are UTC text at second precision: 'YYYY-MM-DDTHH:MM:SSZ'.

CREATE TABLE owner (
    id            INTEGER PRIMARY KEY CHECK (id = 1),
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE admin_sessions (
    id         TEXT PRIMARY KEY,
    token_hash BLOB NOT NULL UNIQUE,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at TEXT NOT NULL
);

CREATE INDEX admin_sessions_expires_at ON admin_sessions (expires_at);

CREATE TABLE fragment_groups (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE fragments (
    id            TEXT PRIMARY KEY,
    group_id      TEXT NOT NULL REFERENCES fragment_groups (id) ON DELETE RESTRICT,
    name          TEXT NOT NULL,
    name_en       TEXT NOT NULL DEFAULT '',
    prompt        TEXT NOT NULL,
    notes         TEXT NOT NULL DEFAULT '',
    source        TEXT NOT NULL DEFAULT '',
    scope         TEXT NOT NULL CHECK (scope IN ('identity', 'pack', 'sticker')),
    primary_color TEXT CHECK (primary_color IS NULL OR (length(primary_color) = 7 AND primary_color GLOB '#[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]')),
    secondary_color TEXT CHECK (secondary_color IS NULL OR (length(secondary_color) = 7 AND secondary_color GLOB '#[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]')),
    accent_color  TEXT CHECK (accent_color IS NULL OR (length(accent_color) = 7 AND accent_color GLOB '#[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]')),
    line_color    TEXT CHECK (line_color IS NULL OR (length(line_color) = 7 AND line_color GLOB '#[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]')),
    archived_at   TEXT,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX fragments_group ON fragments (group_id);
CREATE INDEX fragments_scope ON fragments (scope);

CREATE TABLE reference_assets (
    id              TEXT PRIMARY KEY,
    filename        TEXT NOT NULL,
    content_type    TEXT NOT NULL,
    byte_size       INTEGER NOT NULL CHECK (byte_size > 0),
    width           INTEGER NOT NULL CHECK (width > 0),
    height          INTEGER NOT NULL CHECK (height > 0),
    checksum_sha256 TEXT NOT NULL CHECK (length(checksum_sha256) = 64),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE characters (
    id                 TEXT PRIMARY KEY,
    name               TEXT NOT NULL,
    description        TEXT NOT NULL DEFAULT '',
    identity_notes     TEXT NOT NULL DEFAULT '',
    rights_notes       TEXT NOT NULL DEFAULT '',
    reference_asset_id TEXT REFERENCES reference_assets (id) ON DELETE SET NULL,
    archived_at        TEXT,
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX characters_reference_asset ON characters (reference_asset_id);

CREATE TABLE designs (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    character_id   TEXT REFERENCES characters (id) ON DELETE SET NULL,
    rows           INTEGER NOT NULL CHECK (rows BETWEEN 1 AND 8),
    columns        INTEGER NOT NULL CHECK (columns BETWEEN 1 AND 8),
    outline_width  INTEGER NOT NULL CHECK (outline_width BETWEEN 0 AND 16),
    outline_color  TEXT NOT NULL CHECK (length(outline_color) = 7 AND outline_color GLOB '#[0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f][0-9A-Fa-f]'),
    draft_json     TEXT NOT NULL CHECK (json_valid(draft_json)),
    snapshot_json  TEXT NOT NULL CHECK (json_valid(snapshot_json) AND json_type(snapshot_json, '$.schema_version') = 'integer' AND json_extract(snapshot_json, '$.schema_version') = 1),
    compiled_prompt TEXT NOT NULL DEFAULT '',
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX designs_character ON designs (character_id);

CREATE TABLE packs (
    id                      TEXT PRIMARY KEY,
    slug                    TEXT NOT NULL COLLATE NOCASE UNIQUE,
    name                    TEXT NOT NULL,
    author                  TEXT NOT NULL,
    description             TEXT NOT NULL DEFAULT '',
    design_id               TEXT REFERENCES designs (id) ON DELETE SET NULL,
    status                  TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'ready', 'revoked')),
    current_version         INTEGER NOT NULL DEFAULT 0 CHECK (current_version >= 0),
    source_id               TEXT,
    source_filename         TEXT NOT NULL DEFAULT '',
    source_checksum_sha256  TEXT NOT NULL DEFAULT '',
    source_width            INTEGER NOT NULL DEFAULT 0,
    source_height           INTEGER NOT NULL DEFAULT 0,
    source_byte_size        INTEGER NOT NULL DEFAULT 0,
    created_at              TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at              TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX packs_status_updated ON packs (status, updated_at, id);
CREATE INDEX packs_design ON packs (design_id);

CREATE TABLE pack_tags (
    pack_id TEXT NOT NULL REFERENCES packs (id) ON DELETE CASCADE,
    tag     TEXT NOT NULL,
    PRIMARY KEY (pack_id, tag)
);

CREATE INDEX pack_tags_tag ON pack_tags (tag);

CREATE TABLE pack_versions (
    pack_id          TEXT NOT NULL REFERENCES packs (id) ON DELETE CASCADE,
    version          INTEGER NOT NULL CHECK (version > 0),
    job_id           TEXT NOT NULL UNIQUE,
    checksum_sha256  TEXT NOT NULL CHECK (length(checksum_sha256) = 64),
    sticker_count    INTEGER NOT NULL CHECK (sticker_count > 0),
    manifest_json    TEXT NOT NULL CHECK (json_valid(manifest_json)),
    bundle_path      TEXT NOT NULL,
    bundle_byte_size INTEGER NOT NULL CHECK (bundle_byte_size > 0),
    thumbnail_path   TEXT NOT NULL,
    output_dir       TEXT NOT NULL,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (pack_id, version)
);

CREATE TABLE stickers (
    pack_id         TEXT NOT NULL,
    version         INTEGER NOT NULL,
    sort_order      INTEGER NOT NULL CHECK (sort_order >= 0),
    filename        TEXT NOT NULL,
    object_path     TEXT NOT NULL,
    thumb_path      TEXT NOT NULL,
    checksum_sha256 TEXT NOT NULL CHECK (length(checksum_sha256) = 64),
    width           INTEGER NOT NULL CHECK (width > 0),
    height          INTEGER NOT NULL CHECK (height > 0),
    emoji_hint      TEXT NOT NULL DEFAULT '',
    keywords_json   TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(keywords_json)),
    PRIMARY KEY (pack_id, version, sort_order),
    UNIQUE (pack_id, version, filename),
    FOREIGN KEY (pack_id, version) REFERENCES pack_versions (pack_id, version) ON DELETE CASCADE
);

CREATE TABLE processing_jobs (
    id            TEXT PRIMARY KEY,
    pack_id       TEXT NOT NULL REFERENCES packs (id) ON DELETE CASCADE,
    version       INTEGER NOT NULL CHECK (version > 0),
    status        TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'interrupted')),
    request_json  TEXT NOT NULL CHECK (json_valid(request_json) AND json_type(request_json, '$.schema_version') = 'integer' AND json_extract(request_json, '$.schema_version') = 1),
    result_json   TEXT CHECK (result_json IS NULL OR json_valid(result_json)),
    error_code    TEXT NOT NULL DEFAULT '',
    error_message TEXT NOT NULL DEFAULT '',
    diagnostics   TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    started_at    TEXT,
    finished_at   TEXT,
    UNIQUE (pack_id, version)
);

CREATE UNIQUE INDEX processing_jobs_one_active ON processing_jobs (pack_id) WHERE status IN ('queued', 'running');
CREATE INDEX processing_jobs_status ON processing_jobs (status, created_at);

CREATE TABLE api_keys (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    prefix      TEXT NOT NULL,
    key_hash    BLOB NOT NULL UNIQUE,
    active      INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at  TEXT,
    disabled_at TEXT
);

CREATE INDEX api_keys_active ON api_keys (active);

CREATE TABLE pack_tombstones (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    pack_id         TEXT NOT NULL UNIQUE REFERENCES packs (id) ON DELETE RESTRICT,
    version         INTEGER NOT NULL CHECK (version > 0),
    revoked_at      TEXT NOT NULL,
    reason_code     TEXT NOT NULL,
    checksum_sha256 TEXT NOT NULL CHECK (length(checksum_sha256) = 64)
);

CREATE INDEX pack_tombstones_since ON pack_tombstones (revoked_at, id);

CREATE TABLE seed_versions (
    name            TEXT PRIMARY KEY,
    version         INTEGER NOT NULL,
    checksum_sha256 TEXT NOT NULL DEFAULT '',
    applied_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
