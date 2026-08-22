CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('contributor', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS packs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submitter_id UUID NOT NULL REFERENCES users(id),
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    author TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    current_version INT NOT NULL DEFAULT 1,
    checksum_sha256 TEXT NOT NULL DEFAULT '',
    thumbnail_key TEXT NOT NULL DEFAULT '',
    bundle_key TEXT NOT NULL DEFAULT '',
    sticker_count INT NOT NULL DEFAULT 0,
    ip_policy TEXT NOT NULL DEFAULT 'original_only',
    status TEXT NOT NULL CHECK (status IN (
        'draft', 'processing', 'pending_review', 'awaiting_confirm',
        'rejected', 'published', 'revoked'
    )),
    reject_reason TEXT NOT NULL DEFAULT '',
    source_kind TEXT NOT NULL CHECK (source_kind IN ('sheet', 'files')),
    source_prefix TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS packs_status_updated_idx ON packs (status, updated_at DESC);
CREATE INDEX IF NOT EXISTS packs_tags_idx ON packs USING GIN (tags);

CREATE TABLE IF NOT EXISTS stickers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pack_id UUID NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    emoji_hint TEXT NOT NULL DEFAULT '',
    width INT NOT NULL DEFAULT 512,
    height INT NOT NULL DEFAULT 512,
    object_key TEXT NOT NULL,
    thumb_key TEXT NOT NULL,
    sort_order INT NOT NULL
);

CREATE TABLE IF NOT EXISTS pack_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pack_id UUID NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    version INT NOT NULL,
    checksum_sha256 TEXT NOT NULL,
    bundle_key TEXT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (pack_id, version)
);

CREATE TABLE IF NOT EXISTS pack_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pack_id UUID NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES users(id),
    action TEXT NOT NULL,
    reason_code TEXT NOT NULL DEFAULT '',
    reason_text TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS pack_reviews_reason_code_idx ON pack_reviews (reason_code) WHERE action = 'reject';

CREATE OR REPLACE VIEW view_rejection_analytics AS
SELECT 
    reason_code,
    COUNT(*) AS rejection_count,
    MAX(created_at) AS last_rejected_at
FROM pack_reviews 
WHERE action = 'reject' AND reason_code != '' 
GROUP BY reason_code;

CREATE TABLE IF NOT EXISTS pack_tombstones (
    pack_id UUID PRIMARY KEY REFERENCES packs(id),
    revoked_by UUID NOT NULL REFERENCES users(id),
    reason_code TEXT NOT NULL,
    reason_text TEXT NOT NULL DEFAULT '',
    checksum_sha256 TEXT NOT NULL DEFAULT '',
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS pack_tombstones_revoked_idx ON pack_tombstones (revoked_at);

CREATE TABLE IF NOT EXISTS processing_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pack_id UUID NOT NULL REFERENCES packs(id) ON DELETE CASCADE,
    source_kind TEXT NOT NULL,
    source_prefix TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'done', 'failed')),
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS processing_jobs_status_idx ON processing_jobs (status);
