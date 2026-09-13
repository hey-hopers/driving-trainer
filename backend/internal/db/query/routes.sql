-- name: CreateRoute :one
INSERT INTO routes (
    id,
    source,
    geometry,
    distance_meters,
    duration_seconds,
    original_polyline
) VALUES (
    gen_random_uuid(),
    $1,
    ST_GeogFromText(sqlc.arg(geometry_wkt)::text),
    sqlc.arg(distance_meters),
    sqlc.arg(duration_seconds),
    sqlc.arg(original_polyline)
)
RETURNING
    id::text,
    source,
    ST_AsGeoJSON(geometry::geometry)::text AS geometry_geojson,
    distance_meters,
    duration_seconds,
    original_polyline,
    created_at;

-- name: GetRouteByID :one
SELECT
    id::text,
    source,
    ST_AsGeoJSON(geometry::geometry)::text AS geometry_geojson,
    distance_meters,
    duration_seconds,
    original_polyline,
    created_at
FROM routes
WHERE id = sqlc.arg(id)::uuid;

-- name: CreateRouteSegment :one
INSERT INTO route_segments (
    id,
    route_id,
    sequence,
    geometry,
    distance_meters,
    duration_seconds,
    road_class,
    road_name,
    road_use,
    speed_limit_kph,
    elevation_start_m,
    elevation_end_m,
    incline_avg_percent,
    incline_max_percent
) VALUES (
    gen_random_uuid(),
    sqlc.arg(route_id)::uuid,
    sqlc.arg(sequence),
    ST_GeogFromText(sqlc.arg(geometry_wkt)::text),
    sqlc.arg(distance_meters),
    sqlc.narg(duration_seconds),
    sqlc.narg(road_class),
    sqlc.narg(road_name),
    sqlc.narg(road_use),
    sqlc.narg(speed_limit_kph),
    sqlc.narg(elevation_start_m),
    sqlc.narg(elevation_end_m),
    sqlc.narg(incline_avg_percent),
    sqlc.narg(incline_max_percent)
)
RETURNING
    id::text,
    route_id::text,
    sequence,
    ST_AsGeoJSON(geometry::geometry)::text AS geometry_geojson,
    distance_meters,
    duration_seconds,
    road_class,
    road_name,
    road_use,
    speed_limit_kph,
    elevation_start_m,
    elevation_end_m,
    incline_avg_percent,
    incline_max_percent,
    created_at;

-- name: CreateRouteEvent :one
INSERT INTO route_events (
    id,
    route_id,
    segment_id,
    type,
    position,
    route_distance_meters,
    difficulty_score,
    metadata
) VALUES (
    gen_random_uuid(),
    sqlc.arg(route_id)::uuid,
    sqlc.narg(segment_id)::uuid,
    sqlc.arg(type),
    ST_GeogFromText(sqlc.arg(position_wkt)::text),
    sqlc.narg(route_distance_meters),
    sqlc.narg(difficulty_score),
    sqlc.narg(metadata)
)
RETURNING
    id::text,
    route_id::text,
    COALESCE(segment_id::text, '')::text AS segment_id,
    type,
    ST_AsGeoJSON(position::geometry)::text AS position_geojson,
    route_distance_meters,
    difficulty_score,
    metadata,
    created_at;

-- name: CreateRouteAnalysis :one
INSERT INTO route_analyses (
    id,
    route_id,
    engine_version,
    difficulty_score,
    average_difficulty,
    peak_difficulty,
    complexity_score
) VALUES (
    gen_random_uuid(),
    sqlc.arg(route_id)::uuid,
    sqlc.arg(engine_version),
    sqlc.arg(difficulty_score),
    sqlc.arg(average_difficulty),
    sqlc.arg(peak_difficulty),
    sqlc.arg(complexity_score)
)
RETURNING
    id::text,
    route_id::text,
    engine_version,
    difficulty_score,
    average_difficulty,
    peak_difficulty,
    complexity_score,
    analyzed_at;

-- name: CreateRouteCategoryScore :one
INSERT INTO route_category_scores (
    route_analysis_id,
    category,
    score
) VALUES (
    sqlc.arg(route_analysis_id)::uuid,
    sqlc.arg(category),
    sqlc.arg(score)
)
RETURNING
    route_analysis_id::text,
    category,
    score;

-- name: ListRouteSegments :many
SELECT
    id::text,
    route_id::text,
    sequence,
    ST_AsGeoJSON(geometry::geometry)::text AS geometry_geojson,
    distance_meters,
    duration_seconds,
    road_class,
    road_name,
    road_use,
    speed_limit_kph,
    elevation_start_m,
    elevation_end_m,
    incline_avg_percent,
    incline_max_percent,
    created_at
FROM route_segments
WHERE route_id = sqlc.arg(route_id)::uuid
ORDER BY sequence;

-- name: ListRouteEvents :many
SELECT
    id::text,
    route_id::text,
    COALESCE(segment_id::text, '')::text AS segment_id,
    type,
    ST_AsGeoJSON(position::geometry)::text AS position_geojson,
    route_distance_meters,
    difficulty_score,
    metadata,
    created_at
FROM route_events
WHERE route_id = sqlc.arg(route_id)::uuid
ORDER BY route_distance_meters, created_at;

-- name: GetLatestRouteAnalysis :one
SELECT
    id::text,
    route_id::text,
    engine_version,
    difficulty_score,
    average_difficulty,
    peak_difficulty,
    complexity_score,
    analyzed_at
FROM route_analyses
WHERE route_id = sqlc.arg(route_id)::uuid
ORDER BY analyzed_at DESC
LIMIT 1;

-- name: ListRouteCategoryScores :many
SELECT
    route_analysis_id::text,
    category,
    score
FROM route_category_scores
WHERE route_analysis_id = sqlc.arg(route_analysis_id)::uuid
ORDER BY category;
