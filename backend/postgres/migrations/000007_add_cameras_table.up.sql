CREATE TABLE cameras (
    id SERIAL PRIMARY KEY,
    camera_make VARCHAR(255) NOT NULL,
    camera_type VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (camera_make, camera_type)
);

CREATE INDEX idx_cameras_type ON cameras (camera_type);

CREATE INDEX idx_cameras_make ON cameras (camera_make);
