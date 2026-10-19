-- the older post a duplicate report points at, no foreign key so it outlives a takedown
ALTER TABLE post_reports ADD COLUMN duplicate_of INTEGER;
