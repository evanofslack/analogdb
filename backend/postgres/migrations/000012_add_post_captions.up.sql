CREATE TABLE post_captions (
    post_id INTEGER PRIMARY KEY REFERENCES pictures(id) ON DELETE CASCADE,
    caption TEXT,
    model TEXT NOT NULL,
    version TEXT NOT NULL,
    raw JSONB NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_post_captions_version ON post_captions (version);
