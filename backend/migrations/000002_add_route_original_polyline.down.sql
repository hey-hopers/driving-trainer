DROP INDEX IF EXISTS idx_routes_created_at;

ALTER TABLE routes
DROP COLUMN IF EXISTS original_polyline;
