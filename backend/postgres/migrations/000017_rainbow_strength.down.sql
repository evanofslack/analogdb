DROP INDEX IF EXISTS idx_pictures_rainbow;
ALTER TABLE pictures DROP COLUMN rainbow_key, DROP COLUMN color_strength;
ALTER TABLE pictures
    ADD COLUMN rainbow_key DOUBLE PRECISION GENERATED ALWAYS AS (
        (((id::bigint * 2654435761) % 128) * 360)::float8 +
        CASE
            WHEN color_hue::float8 - 15 + 360 >= 360 THEN color_hue::float8 - 15 + 360 - 360
            ELSE color_hue::float8 - 15 + 360
        END
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_pictures_rainbow ON pictures (rainbow_key, id) WHERE rainbow_key IS NOT NULL;
