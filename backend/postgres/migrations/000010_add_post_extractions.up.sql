CREATE TABLE post_extractions (
    post_id INTEGER PRIMARY KEY REFERENCES pictures(id) ON DELETE CASCADE,
    extractor_version TEXT NOT NULL,
    model TEXT NOT NULL,
    input TEXT NOT NULL,
    input_hash TEXT NOT NULL,
    raw JSONB NOT NULL,
    unmatched JSONB NOT NULL DEFAULT '[]',
    has_unmatched BOOLEAN GENERATED ALWAYS AS (jsonb_array_length(unmatched) > 0) STORED,
    created TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_post_extractions_unmatched ON post_extractions (post_id) WHERE has_unmatched;
