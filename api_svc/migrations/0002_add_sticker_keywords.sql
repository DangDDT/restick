ALTER TABLE stickers ADD COLUMN IF NOT EXISTS keywords TEXT[] NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS stickers_keywords_idx ON stickers USING GIN (keywords);
