CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE routes
ADD COLUMN IF NOT EXISTS original_polyline TEXT;

UPDATE routes
SET original_polyline = ''
WHERE original_polyline IS NULL;

ALTER TABLE routes
ALTER COLUMN original_polyline SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_routes_created_at
ON routes (created_at);
