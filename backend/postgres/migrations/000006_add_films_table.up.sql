BEGIN;

CREATE TABLE films (
    id SERIAL PRIMARY KEY,
    film_make VARCHAR(255),
    film_type VARCHAR(255) NOT NULL,
    film_speed INTEGER,
    color_type VARCHAR(255),
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE ( film_make, film_type, film_speed)
);

CREATE INDEX idx_films_type ON films (film_type);

CREATE INDEX idx_films_make ON films (film_make);

COMMIT;
