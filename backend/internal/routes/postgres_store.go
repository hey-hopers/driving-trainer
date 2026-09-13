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
	segmentIDsBySequence := map[int]string{}
	for _, segment := range route.Segments {
		createdSegment, err := createRouteSegment(ctx, queries, created.ID, segment)
		if err != nil {
			return Route{}, err
		}
		segments = append(segments, createdSegment)
		segmentIDsBySequence[createdSegment.Sequence] = createdSegment.ID
	}
	created.Segments = segments

	events := make([]RouteEvent, 0, len(route.Events))
	for _, event := range route.Events {
		createdEvent, err := createRouteEvent(ctx, queries, created.ID, segmentIDsBySequence, event)
		if err != nil {
			return Route{}, err
		}
		events = append(events, createdEvent)
	}
	created.Events = events

	analysis, err := createRouteAnalysis(ctx, queries, created.ID, route.Analysis)
	if err != nil {
		return Route{}, err
	}
	created.Analysis = analysis

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

	events, err := s.listRouteEvents(ctx, route.ID)
	if err != nil {
		return Route{}, err
	}
	route.Events = events

	analysis, err := s.getLatestRouteAnalysis(ctx, route.ID)
	if err != nil {
		return Route{}, err
	}
	route.Analysis = analysis

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
		RouteID:           parsedRouteID,
		Sequence:          int32(segment.Sequence),
		GeometryWkt:       geometryWKT,
		DistanceMeters:    int32(segment.DistanceMeters),
		DurationSeconds:   nullableInt32(segment.DurationSeconds),
		RoadClass:         nullableText(segment.RoadClass),
		RoadName:          nullableText(segment.RoadName),
		RoadUse:           nullableText(segment.RoadUse),
		SpeedLimitKph:     nullableInt32(segment.SpeedLimitKph),
		ElevationStartM:   nullableFloat64(segment.ElevationStartM),
		ElevationEndM:     nullableFloat64(segment.ElevationEndM),
		InclineAvgPercent: nullableFloat64(segment.InclineAvgPct),
		InclineMaxPercent: nullableFloat64(segment.InclineMaxPct),
	})
	if err != nil {
		return RouteSegment{}, err
	}

	return routeSegmentFromCreateRow(row)
}

func createRouteEvent(ctx context.Context, queries *db.Queries, routeID string, segmentIDsBySequence map[int]string, event RouteEventResult) (RouteEvent, error) {
	parsedRouteID, err := uuid(routeID)
	if err != nil {
		return RouteEvent{}, err
	}

	positionWKT, err := pointWKT(event.Position)
	if err != nil {
		return RouteEvent{}, err
	}

	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return RouteEvent{}, err
	}

	row, err := queries.CreateRouteEvent(ctx, db.CreateRouteEventParams{
		RouteID:             parsedRouteID,
		SegmentID:           nullableUUID(segmentIDsBySequence[event.SegmentSequence]),
		Type:                event.Type,
		PositionWkt:         positionWKT,
		RouteDistanceMeters: nullableInt32(event.RouteDistanceMeters),
		DifficultyScore:     nullableNumeric(event.DifficultyScore),
		Metadata:            metadata,
	})
	if err != nil {
		return RouteEvent{}, err
	}

	return routeEventFromCreateRow(row)
}

func createRouteAnalysis(ctx context.Context, queries *db.Queries, routeID string, analysis RouteAnalysis) (RouteAnalysis, error) {
	parsedRouteID, err := uuid(routeID)
	if err != nil {
		return RouteAnalysis{}, err
	}

	if analysis.EngineVersion == "" {
		analysis.EngineVersion = difficultyEngineVersionV1
	}
	if analysis.OverallDifficulty == 0 && analysis.Difficulty > 0 {
		analysis.OverallDifficulty = analysis.Difficulty
	}
	if analysis.Difficulty == 0 && analysis.OverallDifficulty > 0 {
		analysis.Difficulty = analysis.OverallDifficulty
	}

	row, err := queries.CreateRouteAnalysis(ctx, db.CreateRouteAnalysisParams{
		RouteID:           parsedRouteID,
		EngineVersion:     analysis.EngineVersion,
		DifficultyScore:   numeric(analysis.OverallDifficulty),
		AverageDifficulty: nullableNumeric(analysis.AverageDifficulty),
		PeakDifficulty:    nullableNumeric(analysis.PeakDifficulty),
		ComplexityScore:   nullableNumeric(analysis.ComplexityScore),
	})
	if err != nil {
		return RouteAnalysis{}, err
	}

	categoryScores := analysis.CategoryScores
	if len(categoryScores) == 0 {
		categoryScores = categoryScoresFromLegacy(analysis.Categories)
	}
	routeAnalysisID, err := uuid(row.ID)
	if err != nil {
		return RouteAnalysis{}, err
	}

	for category, score := range categoryScores {
		if _, err := queries.CreateRouteCategoryScore(ctx, db.CreateRouteCategoryScoreParams{
			RouteAnalysisID: routeAnalysisID,
			Category:        category,
			Score:           numeric(score),
		}); err != nil {
			return RouteAnalysis{}, err
		}
	}

	return routeAnalysisFromFields(
		row.EngineVersion,
		row.DifficultyScore,
		row.AverageDifficulty,
		row.PeakDifficulty,
		row.ComplexityScore,
		categoryScores,
	), nil
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

func (s *PostgresStore) listRouteEvents(ctx context.Context, routeID string) ([]RouteEvent, error) {
	parsedRouteID, err := uuid(routeID)
	if err != nil {
		return nil, err
	}

	rows, err := s.queries.ListRouteEvents(ctx, parsedRouteID)
	if err != nil {
		return nil, err
	}

	events := make([]RouteEvent, 0, len(rows))
	for _, row := range rows {
		event, err := routeEventFromListRow(row)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

func (s *PostgresStore) getLatestRouteAnalysis(ctx context.Context, routeID string) (RouteAnalysis, error) {
	parsedRouteID, err := uuid(routeID)
	if err != nil {
		return RouteAnalysis{}, err
	}

	row, err := s.queries.GetLatestRouteAnalysis(ctx, parsedRouteID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RouteAnalysis{}, nil
		}
		return RouteAnalysis{}, err
	}

	routeAnalysisID, err := uuid(row.ID)
	if err != nil {
		return RouteAnalysis{}, err
	}

	categoryRows, err := s.queries.ListRouteCategoryScores(ctx, routeAnalysisID)
	if err != nil {
		return RouteAnalysis{}, err
	}

	categoryScores := make(map[string]float64, len(categoryRows))
	for _, categoryRow := range categoryRows {
		categoryScores[categoryRow.Category] = numericOrZero(categoryRow.Score)
	}

	return routeAnalysisFromFields(
		row.EngineVersion,
		row.DifficultyScore,
		row.AverageDifficulty,
		row.PeakDifficulty,
		row.ComplexityScore,
		categoryScores,
	), nil
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
		ElevationStartM: floatPtrOrNil(row.ElevationStartM),
		ElevationEndM:   floatPtrOrNil(row.ElevationEndM),
		InclineAvgPct:   floatPtrOrNil(row.InclineAvgPercent),
		InclineMaxPct:   floatPtrOrNil(row.InclineMaxPercent),
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
		ElevationStartM: floatPtrOrNil(row.ElevationStartM),
		ElevationEndM:   floatPtrOrNil(row.ElevationEndM),
		InclineAvgPct:   floatPtrOrNil(row.InclineAvgPercent),
		InclineMaxPct:   floatPtrOrNil(row.InclineMaxPercent),
		CreatedAt:       row.CreatedAt.Time,
	}, nil
}

func routeEventFromCreateRow(row db.CreateRouteEventRow) (RouteEvent, error) {
	return routeEventFromFields(
		row.ID,
		row.RouteID,
		row.SegmentID,
		row.Type,
		row.PositionGeojson,
		row.RouteDistanceMeters,
		row.DifficultyScore,
		row.Metadata,
		row.CreatedAt,
	)
}

func routeEventFromListRow(row db.ListRouteEventsRow) (RouteEvent, error) {
	return routeEventFromFields(
		row.ID,
		row.RouteID,
		row.SegmentID,
		row.Type,
		row.PositionGeojson,
		row.RouteDistanceMeters,
		row.DifficultyScore,
		row.Metadata,
		row.CreatedAt,
	)
}

func routeEventFromFields(
	id string,
	routeID string,
	segmentID string,
	eventType string,
	positionGeoJSON string,
	routeDistanceMeters pgtype.Int4,
	difficultyScore pgtype.Numeric,
	metadataBytes []byte,
	createdAt pgtype.Timestamptz,
) (RouteEvent, error) {
	position, err := coordinateFromPointGeoJSON(positionGeoJSON)
	if err != nil {
		return RouteEvent{}, err
	}

	metadata := map[string]any{}
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return RouteEvent{}, fmt.Errorf("decode route event metadata: %w", err)
		}
	}

	return RouteEvent{
		ID:                  id,
		RouteID:             routeID,
		SegmentID:           segmentID,
		Type:                eventType,
		Position:            position,
		RouteDistanceMeters: intOrZero(routeDistanceMeters),
		DifficultyScore:     numericOrZero(difficultyScore),
		Metadata:            metadata,
		CreatedAt:           createdAt.Time,
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

func pointWKT(coordinate Coordinate) (string, error) {
	if coordinate.Latitude < -90 || coordinate.Latitude > 90 ||
		coordinate.Longitude < -180 || coordinate.Longitude > 180 {
		return "", errors.New("route event position contains invalid coordinate")
	}

	return "POINT(" + formatFloat(coordinate.Longitude) + " " + formatFloat(coordinate.Latitude) + ")", nil
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

type lineStringGeoJSON struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

type pointGeoJSON struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
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

func coordinateFromPointGeoJSON(value string) (Coordinate, error) {
	var geometry pointGeoJSON
	if err := json.Unmarshal([]byte(value), &geometry); err != nil {
		return Coordinate{}, fmt.Errorf("decode route event position geojson: %w", err)
	}
	if geometry.Type != "Point" {
		return Coordinate{}, fmt.Errorf("expected Point geometry, got %q", geometry.Type)
	}
	if len(geometry.Coordinates) != 2 {
		return Coordinate{}, errors.New("route event position must contain longitude and latitude")
	}

	return Coordinate{
		Longitude: geometry.Coordinates[0],
		Latitude:  geometry.Coordinates[1],
	}, nil
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

func nullableUUID(value string) pgtype.UUID {
	if value == "" {
		return pgtype.UUID{}
	}
	id, err := uuid(value)
	if err != nil {
		return pgtype.UUID{}
	}
	return id
}

func nullableFloat64(value *float64) pgtype.Float8 {
	if value == nil {
		return pgtype.Float8{}
	}
	return pgtype.Float8{Float64: *value, Valid: true}
}

func nullableNumeric(value float64) pgtype.Numeric {
	if value == 0 {
		return pgtype.Numeric{}
	}
	var numeric pgtype.Numeric
	if err := numeric.Scan(formatFloat(value)); err != nil {
		return pgtype.Numeric{}
	}
	return numeric
}

func numeric(value float64) pgtype.Numeric {
	var numeric pgtype.Numeric
	if err := numeric.Scan(formatFloat(value)); err != nil {
		return pgtype.Numeric{}
	}
	return numeric
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

func floatPtrOrNil(value pgtype.Float8) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func numericOrZero(value pgtype.Numeric) float64 {
	number, err := value.Float64Value()
	if err != nil || !number.Valid {
		return 0
	}
	return number.Float64
}

func routeAnalysisFromFields(
	engineVersion string,
	difficultyScore pgtype.Numeric,
	averageDifficulty pgtype.Numeric,
	peakDifficulty pgtype.Numeric,
	complexityScore pgtype.Numeric,
	categoryScores map[string]float64,
) RouteAnalysis {
	overallDifficulty := numericOrZero(difficultyScore)
	categories := legacyCategoriesFromScores(categoryScores)
	return RouteAnalysis{
		EngineVersion:     engineVersion,
		Difficulty:        overallDifficulty,
		OverallDifficulty: overallDifficulty,
		AverageDifficulty: numericOrZero(averageDifficulty),
		PeakDifficulty:    numericOrZero(peakDifficulty),
		ComplexityScore:   numericOrZero(complexityScore),
		CategoryScores:    categoryScores,
		Categories:        categories,
	}
}

func legacyCategoriesFromScores(scores map[string]float64) RouteCategoryScores {
	return RouteCategoryScores{
		Hills:         scores["hills"],
		Curves:        scores["curves"],
		Intersections: scores["intersections"],
		HighSpeed:     scores["highSpeed"],
	}
}

func categoryScoresFromLegacy(categories RouteCategoryScores) map[string]float64 {
	return map[string]float64{
		"hills":         categories.Hills,
		"curves":        categories.Curves,
		"intersections": categories.Intersections,
		"highSpeed":     categories.HighSpeed,
	}
}
