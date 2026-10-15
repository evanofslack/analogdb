-- the rainbow key is computed by the API, so its tiers can be retuned without a migration
DROP INDEX IF EXISTS idx_pictures_rainbow;
ALTER TABLE pictures DROP COLUMN rainbow_key;
ALTER TABLE pictures
    ADD COLUMN color_strength REAL,
    ADD COLUMN rainbow_key DOUBLE PRECISION;

COMMENT ON COLUMN pictures.color_strength IS 'percent times chroma of the representative color, or the dominant neutral share of a black, gray or white post';

CREATE INDEX IF NOT EXISTS idx_pictures_rainbow ON pictures (rainbow_key, id) WHERE rainbow_key IS NOT NULL;
