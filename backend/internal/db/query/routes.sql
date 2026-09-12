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
