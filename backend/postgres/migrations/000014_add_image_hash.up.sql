-- image_hash is a 64 bit dHash of the low res image, used to find reposts
ALTER TABLE pictures ADD COLUMN image_hash BIGINT;
