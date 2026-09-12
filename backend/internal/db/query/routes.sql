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
    speed_limit_kph
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
    sqlc.narg(speed_limit_kph)
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
    created_at;

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
    created_at
FROM route_segments
WHERE route_id = sqlc.arg(route_id)::uuid
ORDER BY sequence;
