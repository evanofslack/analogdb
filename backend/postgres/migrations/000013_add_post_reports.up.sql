-- removed_posts keeps a deleted post's permalink so the scraper never adds it again
CREATE TABLE removed_posts (
    post_id INTEGER PRIMARY KEY,
    permalink TEXT UNIQUE NOT NULL,
    title TEXT,
    author TEXT,
    url TEXT NOT NULL,
    low_url TEXT,
    med_url TEXT,
    high_url TEXT,
    posted_at TIMESTAMP WITH TIME ZONE,
    reason TEXT,
    removed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- no foreign key to pictures, a report outlives the post it took down
CREATE TABLE post_reports (
    id BIGSERIAL PRIMARY KEY,
    post_id INTEGER NOT NULL,
    reason TEXT NOT NULL,
    message TEXT,
    email TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE
);

CREATE INDEX idx_post_reports_post ON post_reports (post_id);
CREATE INDEX idx_post_reports_open ON post_reports (id DESC) WHERE resolved_at IS NULL;
