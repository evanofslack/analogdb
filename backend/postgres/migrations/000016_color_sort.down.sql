DROP INDEX IF EXISTS idx_pictures_rainbow;
ALTER TABLE pictures DROP COLUMN rainbow_key, DROP COLUMN color_hue, DROP COLUMN color_chroma;
ALTER TABLE colors DROP COLUMN oklab_l, DROP COLUMN oklab_a, DROP COLUMN oklab_b, DROP COLUMN chroma, DROP COLUMN hue;
