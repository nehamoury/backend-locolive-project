CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE places (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    address     TEXT,
    geom        GEOMETRY(Point, 4326),
    post_count  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_places_slug ON places(slug);
CREATE INDEX idx_places_geom ON places USING GIST(geom);
CREATE INDEX idx_places_name_trgm ON places USING GIN (name gin_trgm_ops);
