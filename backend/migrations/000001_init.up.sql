CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE routes (
    id UUID PRIMARY KEY,
    name VARCHAR(150),
    source VARCHAR(30) NOT NULL,
    geometry GEOGRAPHY(LINESTRING, 4326) NOT NULL,
    distance_meters INTEGER NOT NULL,
    duration_seconds INTEGER NOT NULL,
    difficulty_score NUMERIC(4,2),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_routes_geometry
ON routes USING GIST (geometry);

CREATE TABLE route_segments (
    id UUID PRIMARY KEY,
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,

    geometry GEOGRAPHY(LINESTRING, 4326) NOT NULL,

    distance_meters INTEGER NOT NULL,
    duration_seconds INTEGER,

    road_class VARCHAR(50),
    road_name VARCHAR(255),

    speed_limit_kph INTEGER,

    elevation_start_m DOUBLE PRECISION,
    elevation_end_m DOUBLE PRECISION,

    incline_avg_percent DOUBLE PRECISION,
    incline_max_percent DOUBLE PRECISION,

    curve_score NUMERIC(4,2),
    difficulty_score NUMERIC(4,2),

    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(route_id, sequence)
);

CREATE INDEX idx_route_segments_geometry
ON route_segments USING GIST (geometry);

CREATE TABLE route_events (
    id UUID PRIMARY KEY,
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    segment_id UUID REFERENCES route_segments(id) ON DELETE SET NULL,

    type VARCHAR(50) NOT NULL,

    position GEOGRAPHY(POINT, 4326) NOT NULL,

    route_distance_meters INTEGER,

    difficulty_score NUMERIC(4,2),

    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_route_events_position
ON route_events USING GIST (position);

CREATE TABLE route_analyses (
    id UUID PRIMARY KEY,
    route_id UUID NOT NULL REFERENCES routes(id) ON DELETE CASCADE,

    engine_version VARCHAR(20) NOT NULL,

    difficulty_score NUMERIC(4,2) NOT NULL,
    average_difficulty NUMERIC(4,2),
    peak_difficulty NUMERIC(4,2),
    complexity_score NUMERIC(4,2),

    analyzed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE route_category_scores (
    route_analysis_id UUID NOT NULL
        REFERENCES route_analyses(id)
        ON DELETE CASCADE,

    category VARCHAR(50) NOT NULL,
    score NUMERIC(4,2) NOT NULL,

    PRIMARY KEY(route_analysis_id, category)
);