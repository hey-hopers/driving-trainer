package routes

import (
	"context"
	"errors"
)

type Router interface {
	CalculateRoute(ctx context.Context, origin Coordinate, destination Coordinate) (RouteResult, error)
}

type RouteResult struct {
	DistanceMeters  int
	DurationSeconds int
	Polyline        string
}

type Service struct {
	router Router
}

func NewService(router Router) *Service {
	return &Service{router: router}
}

func (s *Service) Analyze(ctx context.Context, req AnalyzeRouteRequest) (AnalyzeRouteResponse, error) {
	if s.router == nil {
		return AnalyzeRouteResponse{}, errors.New("router is required")
	}

	route, err := s.router.CalculateRoute(ctx, *req.Origin, *req.Destination)
	if err != nil {
		return AnalyzeRouteResponse{}, err
	}

	return AnalyzeRouteResponse{
		Route: RouteSummary{
			DistanceMeters:  route.DistanceMeters,
			DurationSeconds: route.DurationSeconds,
			Polyline:        route.Polyline,
		},
		Analysis: RouteAnalysis{
			Difficulty: 0,
			Categories: RouteCategoryScores{},
		},
		Events: []any{},
	}, nil
}
