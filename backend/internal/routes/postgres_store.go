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
	queries *db.Queries
}

func NewPostgresStore(queries *db.Queries) *PostgresStore {
	return &PostgresStore{queries: queries}
}

func (s *PostgresStore) CreateRoute(ctx context.Context, route NewRoute) (Route, error) {
	if s.queries == nil {
		return Route{}, errors.New("queries are required")
	}

	geometryWKT, err := lineStringWKT(route.Geometry)
	if err != nil {
		return Route{}, err
	}

	row, err := s.queries.CreateRoute(ctx, db.CreateRouteParams{
		Source:           route.Source,
		GeometryWkt:      geometryWKT,
		DistanceMeters:   int32(route.DistanceMeters),
		DurationSeconds:  int32(route.DurationSeconds),
		OriginalPolyline: route.Polyline,
	})
	if err != nil {
		return Route{}, err
	}

	return routeFromCreateRow(row)
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

	return routeFromGetRow(row)
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
