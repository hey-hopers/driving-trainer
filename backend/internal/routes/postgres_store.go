package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	db "driving-trainer/backend/internal/db/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresStore struct {
	db      db.DBTX
	queries *db.Queries
}

func NewPostgresStore(database db.DBTX) *PostgresStore {
	return &PostgresStore{
		db:      database,
		queries: db.New(database),
	}
}

func (s *PostgresStore) CreateRoute(ctx context.Context, route NewRoute) (Route, error) {
	if s.queries == nil {
		return Route{}, errors.New("queries are required")
	}
	if beginner, ok := s.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	}); ok {
		tx, err := beginner.Begin(ctx)
		if err != nil {
			return Route{}, err
		}
		queries := s.queries.WithTx(tx)
		created, err := createRouteWithSegments(ctx, queries, route)
		if err != nil {
			_ = tx.Rollback(ctx)
			return Route{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return Route{}, err
		}
		return created, nil
	}

	return createRouteWithSegments(ctx, s.queries, route)
}

func createRouteWithSegments(ctx context.Context, queries *db.Queries, route NewRoute) (Route, error) {
	geometryWKT, err := lineStringWKT(route.Geometry)
	if err != nil {
		return Route{}, err
	}

	row, err := queries.CreateRoute(ctx, db.CreateRouteParams{
		Source:           route.Source,
		GeometryWkt:      geometryWKT,
		DistanceMeters:   int32(route.DistanceMeters),
		DurationSeconds:  int32(route.DurationSeconds),
		OriginalPolyline: route.Polyline,
	})
	if err != nil {
		return Route{}, err
	}

	created, err := routeFromCreateRow(row)
	if err != nil {
		return Route{}, err
	}

	segments := make([]RouteSegment, 0, len(route.Segments))
	for _, segment := range route.Segments {
		createdSegment, err := createRouteSegment(ctx, queries, created.ID, segment)
		if err != nil {
			return Route{}, err
		}
		segments = append(segments, createdSegment)
	}
	created.Segments = segments

	return created, nil
}

func (s *PostgresStore) GetRoute(ctx context.Context, id string) (Route, error) {
	if s.queries == nil {
		return Route{}, errors.New("queries are required")
	}

	var routeID pgtype.UUID
	if err := routeID.Scan(id); err != nil {
		return Route{}, ErrInvalidRouteID
	}

	row, err := s.queries.GetRouteByID(ctx, routeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Route{}, ErrRouteNotFound
		}
		return Route{}, err
	}

	route, err := routeFromGetRow(row)
	if err != nil {
		return Route{}, err
	}

	segments, err := s.listRouteSegments(ctx, route.ID)
	if err != nil {
		return Route{}, err
	}
	route.Segments = segments

	return route, nil
}

func createRouteSegment(ctx context.Context, queries *db.Queries, routeID string, segment RouteSegmentResult) (RouteSegment, error) {
	geometryWKT, err := lineStringWKT(segment.Geometry)
	if err != nil {
		return RouteSegment{}, err
	}

	parsedRouteID, err := uuid(routeID)
	if err != nil {
		return RouteSegment{}, err
	}

	row, err := queries.CreateRouteSegment(ctx, db.CreateRouteSegmentParams{
		RouteID:         parsedRouteID,
		Sequence:        int32(segment.Sequence),
		GeometryWkt:     geometryWKT,
		DistanceMeters:  int32(segment.DistanceMeters),
		DurationSeconds: nullableInt32(segment.DurationSeconds),
		RoadClass:       nullableText(segment.RoadClass),
		RoadName:        nullableText(segment.RoadName),
		RoadUse:         nullableText(segment.RoadUse),
		SpeedLimitKph:   nullableInt32(segment.SpeedLimitKph),
	})
	if err != nil {
		return RouteSegment{}, err
	}

	return routeSegmentFromCreateRow(row)
}

func (s *PostgresStore) listRouteSegments(ctx context.Context, routeID string) ([]RouteSegment, error) {
	parsedRouteID, err := uuid(routeID)
	if err != nil {
		return nil, err
	}

	rows, err := s.queries.ListRouteSegments(ctx, parsedRouteID)
	if err != nil {
		return nil, err
	}

	segments := make([]RouteSegment, 0, len(rows))
	for _, row := range rows {
		segment, err := routeSegmentFromListRow(row)
		if err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}

	return segments, nil
}

func routeFromCreateRow(row db.CreateRouteRow) (Route, error) {
	geometry, err := coordinatesFromGeoJSON(row.GeometryGeojson)
	if err != nil {
		return Route{}, err
	}

	return Route{
		ID:              row.ID,
		Source:          row.Source,
		Geometry:        geometry,
		DistanceMeters:  int(row.DistanceMeters),
		DurationSeconds: int(row.DurationSeconds),
		Polyline:        row.OriginalPolyline,
		CreatedAt:       row.CreatedAt.Time,
	}, nil
}

func routeFromGetRow(row db.GetRouteByIDRow) (Route, error) {
	geometry, err := coordinatesFromGeoJSON(row.GeometryGeojson)
	if err != nil {
		return Route{}, err
	}

	return Route{
		ID:              row.ID,
		Source:          row.Source,
		Geometry:        geometry,
		DistanceMeters:  int(row.DistanceMeters),
		DurationSeconds: int(row.DurationSeconds),
		Polyline:        row.OriginalPolyline,
		CreatedAt:       row.CreatedAt.Time,
	}, nil
}

func routeSegmentFromCreateRow(row db.CreateRouteSegmentRow) (RouteSegment, error) {
	geometry, err := coordinatesFromGeoJSON(row.GeometryGeojson)
	if err != nil {
		return RouteSegment{}, err
	}

	return RouteSegment{
		ID:              row.ID,
		RouteID:         row.RouteID,
		Sequence:        int(row.Sequence),
		Geometry:        geometry,
		DistanceMeters:  int(row.DistanceMeters),
		DurationSeconds: intOrZero(row.DurationSeconds),
		RoadClass:       textOrEmpty(row.RoadClass),
		RoadName:        textOrEmpty(row.RoadName),
		RoadUse:         textOrEmpty(row.RoadUse),
		SpeedLimitKph:   intOrZero(row.SpeedLimitKph),
		CreatedAt:       row.CreatedAt.Time,
	}, nil
}

func routeSegmentFromListRow(row db.ListRouteSegmentsRow) (RouteSegment, error) {
	geometry, err := coordinatesFromGeoJSON(row.GeometryGeojson)
	if err != nil {
		return RouteSegment{}, err
	}

	return RouteSegment{
		ID:              row.ID,
		RouteID:         row.RouteID,
		Sequence:        int(row.Sequence),
		Geometry:        geometry,
		DistanceMeters:  int(row.DistanceMeters),
		DurationSeconds: intOrZero(row.DurationSeconds),
		RoadClass:       textOrEmpty(row.RoadClass),
		RoadName:        textOrEmpty(row.RoadName),
		RoadUse:         textOrEmpty(row.RoadUse),
		SpeedLimitKph:   intOrZero(row.SpeedLimitKph),
		CreatedAt:       row.CreatedAt.Time,
	}, nil
}

func lineStringWKT(coordinates []Coordinate) (string, error) {
	if len(coordinates) < 2 {
		return "", errors.New("route geometry must contain at least two points")
	}

	points := make([]string, 0, len(coordinates))
	for _, coordinate := range coordinates {
		if coordinate.Latitude < -90 || coordinate.Latitude > 90 ||
			coordinate.Longitude < -180 || coordinate.Longitude > 180 {
			return "", errors.New("route geometry contains invalid coordinate")
		}
		points = append(points, formatFloat(coordinate.Longitude)+" "+formatFloat(coordinate.Latitude))
	}

	return "LINESTRING(" + strings.Join(points, ",") + ")", nil
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

type lineStringGeoJSON struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

func coordinatesFromGeoJSON(value string) ([]Coordinate, error) {
	var geometry lineStringGeoJSON
	if err := json.Unmarshal([]byte(value), &geometry); err != nil {
		return nil, fmt.Errorf("decode route geometry geojson: %w", err)
	}
	if geometry.Type != "LineString" {
		return nil, fmt.Errorf("expected LineString geometry, got %q", geometry.Type)
	}
	if len(geometry.Coordinates) < 2 {
		return nil, errors.New("route geometry must contain at least two points")
	}

	coordinates := make([]Coordinate, 0, len(geometry.Coordinates))
	for _, point := range geometry.Coordinates {
		if len(point) != 2 {
			return nil, errors.New("route geometry point must contain longitude and latitude")
		}
		coordinates = append(coordinates, Coordinate{
			Longitude: point[0],
			Latitude:  point[1],
		})
	}

	return coordinates, nil
}

func uuid(value string) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(value); err != nil {
		return pgtype.UUID{}, err
	}
	return id, nil
}

func nullableInt32(value int) pgtype.Int4 {
	if value == 0 {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(value), Valid: true}
}

func nullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func intOrZero(value pgtype.Int4) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int32)
}

func textOrEmpty(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
