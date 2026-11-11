-- Design sheets (uploaded AI-generated sheet images per design) and per-key pack grants.
-- Not live: packs.source_* data is discarded, no backfill.
-- design_sheets.id is also the directory name under sources/.

CREATE TABLE design_sheets (
    id              TEXT PRIMARY KEY,
    design_id       TEXT NOT NULL REFERENCES designs (id) ON DELETE RESTRICT,
    provider        TEXT NOT NULL CHECK (length(provider) BETWEEN 1 AND 32 AND provider NOT GLOB '*[^a-z0-9_-]*'),
    note            TEXT NOT NULL DEFAULT '',
    prompt_snapshot TEXT NOT NULL,
    filename        TEXT NOT NULL,
    media_type      TEXT NOT NULL,
    byte_size       INTEGER NOT NULL CHECK (byte_size > 0),
    width           INTEGER NOT NULL CHECK (width > 0),
    height          INTEGER NOT NULL CHECK (height > 0),
    checksum_sha256 TEXT NOT NULL CHECK (length(checksum_sha256) = 64),
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (design_id, checksum_sha256)
);

CREATE INDEX design_sheets_design ON design_sheets (design_id);

ALTER TABLE packs ADD COLUMN sheet_id TEXT REFERENCES design_sheets (id) ON DELETE RESTRICT;
ALTER TABLE packs DROP COLUMN source_id;
ALTER TABLE packs DROP COLUMN source_filename;
ALTER TABLE packs DROP COLUMN source_checksum_sha256;
ALTER TABLE packs DROP COLUMN source_width;
ALTER TABLE packs DROP COLUMN source_height;
ALTER TABLE packs DROP COLUMN source_byte_size;

CREATE TABLE api_key_packs (
    key_id  TEXT NOT NULL REFERENCES api_keys (id) ON DELETE CASCADE,
    pack_id TEXT NOT NULL REFERENCES packs (id) ON DELETE CASCADE,
    PRIMARY KEY (key_id, pack_id)
);

CREATE INDEX api_key_packs_pack ON api_key_packs (pack_id);

CREATE TABLE api_key_pack_removals (
    key_id     TEXT NOT NULL REFERENCES api_keys (id) ON DELETE CASCADE,
    pack_id    TEXT NOT NULL REFERENCES packs (id) ON DELETE CASCADE,
    removed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (key_id, pack_id)
);
