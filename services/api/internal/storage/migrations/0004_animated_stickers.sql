-- Animated sticker drafts: one clip per (pack, sticker index), processed on its
-- own queue (status column) so clips never consume a processing_jobs version.
-- static_checksum_sha256 is the static sticker the draft was made against; a
-- different checksum in the current version marks the draft stale.

CREATE TABLE sticker_animations (
    id                       TEXT PRIMARY KEY,
    pack_id                  TEXT NOT NULL REFERENCES packs (id) ON DELETE CASCADE,
    sticker_index            INTEGER NOT NULL CHECK (sticker_index >= 0),
    static_checksum_sha256   TEXT NOT NULL CHECK (length(static_checksum_sha256) = 64),
    motion_text              TEXT NOT NULL DEFAULT '',
    clip_id                  TEXT NOT NULL,
    clip_path                TEXT NOT NULL,
    clip_byte_size           INTEGER NOT NULL CHECK (clip_byte_size > 0),
    clip_duration_ms         INTEGER NOT NULL DEFAULT 0,
    clip_checksum_sha256     TEXT NOT NULL CHECK (length(clip_checksum_sha256) = 64),
    status                   TEXT NOT NULL CHECK (status IN ('queued', 'running', 'ready', 'failed', 'interrupted')),
    animated_path            TEXT NOT NULL DEFAULT '',
    animated_checksum_sha256 TEXT NOT NULL DEFAULT '',
    animated_byte_size       INTEGER NOT NULL DEFAULT 0,
    frames                   INTEGER NOT NULL DEFAULT 0,
    fps                      INTEGER NOT NULL DEFAULT 0,
    duration_ms              INTEGER NOT NULL DEFAULT 0,
    quality                  INTEGER NOT NULL DEFAULT 0,
    width                    INTEGER NOT NULL DEFAULT 0,
    height                   INTEGER NOT NULL DEFAULT 0,
    error_code               TEXT NOT NULL DEFAULT '',
    error_message            TEXT NOT NULL DEFAULT '',
    use_animated             INTEGER NOT NULL DEFAULT 1 CHECK (use_animated IN (0, 1)),
    created_at               TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at               TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    started_at               TEXT,
    finished_at              TEXT,
    UNIQUE (pack_id, sticker_index)
);

CREATE INDEX sticker_animations_status ON sticker_animations (status, created_at);

ALTER TABLE stickers ADD COLUMN animated_path TEXT NOT NULL DEFAULT '';
ALTER TABLE stickers ADD COLUMN animated_checksum_sha256 TEXT NOT NULL DEFAULT '';
ALTER TABLE stickers ADD COLUMN animated_byte_size INTEGER NOT NULL DEFAULT 0;
ALTER TABLE stickers ADD COLUMN animated_frames INTEGER NOT NULL DEFAULT 0;
ALTER TABLE stickers ADD COLUMN animated_duration_ms INTEGER NOT NULL DEFAULT 0;
