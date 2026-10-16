CREATE TABLE IF NOT EXISTS stats_themes (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    hidden BOOLEAN NOT NULL DEFAULT false,
    role TEXT,
    center REAL[] NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stats_prompts (
    key TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    text TEXT NOT NULL,
    vector REAL[] NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS stats_seed (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    version TEXT NOT NULL,
    model TEXT NOT NULL,
    updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE stats_themes IS 'visual theme centers in CLIP space, fit offline and named by hand';
COMMENT ON TABLE stats_prompts IS 'CLIP text vectors for the style prompts';
