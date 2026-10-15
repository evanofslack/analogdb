-- derived OKLab and OKLCH values per color cluster, written by the API
ALTER TABLE colors
    ADD COLUMN oklab_l REAL,
    ADD COLUMN oklab_a REAL,
    ADD COLUMN oklab_b REAL,
    ADD COLUMN chroma REAL,
    ADD COLUMN hue REAL;

-- representative color of a post and its place in the rainbow sort
ALTER TABLE pictures
    ADD COLUMN color_hue REAL,
    ADD COLUMN color_chroma REAL,
    ADD COLUMN rainbow_key DOUBLE PRECISION GENERATED ALWAYS AS (
        (((id::bigint * 2654435761) % 128) * 360)::float8 +
        CASE
            WHEN color_hue::float8 - 15 + 360 >= 360 THEN color_hue::float8 - 15 + 360 - 360
            ELSE color_hue::float8 - 15 + 360
        END
    ) STORED;

CREATE INDEX IF NOT EXISTS idx_pictures_rainbow ON pictures (rainbow_key, id) WHERE rainbow_key IS NOT NULL;
