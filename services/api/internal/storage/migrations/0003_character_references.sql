-- Character reference lists: one ordered set of private reference images per
-- character. Existing single references become position 0, then the singular
-- index and column go away. Runs in one migration transaction.

CREATE TABLE character_reference_assets (
    character_id TEXT NOT NULL REFERENCES characters (id) ON DELETE CASCADE,
    asset_id     TEXT NOT NULL REFERENCES reference_assets (id) ON DELETE RESTRICT,
    position     INTEGER NOT NULL CHECK (position >= 0),
    PRIMARY KEY (character_id, asset_id),
    UNIQUE (character_id, position)
);

INSERT INTO character_reference_assets (character_id, asset_id, position)
SELECT id, reference_asset_id, 0 FROM characters WHERE reference_asset_id IS NOT NULL;

DROP INDEX characters_reference_asset;
ALTER TABLE characters DROP COLUMN reference_asset_id;
