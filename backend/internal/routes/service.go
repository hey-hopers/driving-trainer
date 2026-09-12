package routes

import (
	"context"
	"errors"
	"fmt"
)

type Router interface {
	CalculateRoute(ctx context.Context, origin Coordinate, destination Coordinate) (RouteResult, error)
}

type Store interface {
	CreateRoute(ctx context.Context, route NewRoute) (Route, error)
	GetRoute(ctx context.Context, id string) (Route, error)
}

type RouteResult struct {
	DistanceMeters  int
	DurationSeconds int
	Polyline        string
	Segments        []RouteSegmentResult
}

type NewRoute struct {
	Source          string
	Geometry        []Coordinate
	DistanceMeters  int
	DurationSeconds int
	Polyline        string
	Segments        []RouteSegmentResult
}

var (
	ErrRoutePersistence = errors.New("route persistence failed")
	ErrRouteNotFound    = errors.New("route not found")
	ErrInvalidRouteID   = errors.New("invalid route id")
)

type Service struct {
	router Router
	store  Store
}

func NewService(router Router, store Store) *Service {
	return &Service{router: router, store: store}
}

func (s *Service) Analyze(ctx context.Context, req AnalyzeRouteRequest) (AnalyzeRouteResponse, error) {
	if s.router == nil {
		return AnalyzeRouteResponse{}, errors.New("router is required")
	}
	if s.store == nil {
		return AnalyzeRouteResponse{}, errors.New("route store is required")
	}

	route, err := s.router.CalculateRoute(ctx, *req.Origin, *req.Destination)
	if err != nil {
		return AnalyzeRouteResponse{}, err
	}

	coordinates, err := DecodePolyline6(route.Polyline)
	if err != nil {
		return AnalyzeRouteResponse{}, fmt.Errorf("decode route polyline6: %w", err)
	}

	persistedRoute, err := s.store.CreateRoute(ctx, NewRoute{
		Source:          "valhalla",
		Geometry:        coordinates,
		DistanceMeters:  route.DistanceMeters,
		DurationSeconds: route.DurationSeconds,
		Polyline:        route.Polyline,
		Segments:        segmentsOrWholeRoute(route, coordinates),
	})
	if err != nil {
		return AnalyzeRouteResponse{}, fmt.Errorf("%w: %v", ErrRoutePersistence, err)
	}

	return AnalyzeRouteResponse{
		Route: RouteSummary{
			ID:              persistedRoute.ID,
			DistanceMeters:  route.DistanceMeters,
			DurationSeconds: route.DurationSeconds,
			Polyline:        route.Polyline,
			Segments:        segmentResponses(persistedRoute.Segments),
		},
		Analysis: RouteAnalysis{
			Difficulty: 0,
			Categories: RouteCategoryScores{},
		},
		Events: []any{},
	}, nil
}

func (s *Service) Get(ctx context.Context, id string) (GetRouteResponse, error) {
	if !IsValidUUID(id) {
		return GetRouteResponse{}, ErrInvalidRouteID
	}
	if s.store == nil {
		return GetRouteResponse{}, errors.New("route store is required")
	}

	route, err := s.store.GetRoute(ctx, id)
	if err != nil {
		return GetRouteResponse{}, err
	}

	return GetRouteResponse{
		Route: PersistedRouteResponse{
			ID:              route.ID,
			Source:          route.Source,
			Geometry:        route.Geometry,
			DistanceMeters:  route.DistanceMeters,
			DurationSeconds: route.DurationSeconds,
			Polyline:        route.Polyline,
			Segments:        segmentResponses(route.Segments),
			CreatedAt:       route.CreatedAt,
		},
	}, nil
}

func segmentsOrWholeRoute(route RouteResult, geometry []Coordinate) []RouteSegmentResult {
	if len(route.Segments) > 0 {
		return route.Segments
	}

	return []RouteSegmentResult{
		{
			Sequence:        0,
			Geometry:        geometry,
			DistanceMeters:  route.DistanceMeters,
			DurationSeconds: route.DurationSeconds,
		},
	}
}

func segmentResponses(segments []RouteSegment) []RouteSegmentResponse {
	responses := make([]RouteSegmentResponse, 0, len(segments))
	for _, segment := range segments {
		responses = append(responses, RouteSegmentResponse{
			ID:              segment.ID,
			Sequence:        segment.Sequence,
			Geometry:        segment.Geometry,
			DistanceMeters:  segment.DistanceMeters,
			DurationSeconds: segment.DurationSeconds,
			RoadClass:       segment.RoadClass,
			RoadName:        segment.RoadName,
			RoadUse:         segment.RoadUse,
			SpeedLimitKph:   segment.SpeedLimitKph,
		})
	}
	return responses
}
