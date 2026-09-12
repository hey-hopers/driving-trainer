package routes

import "context"

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Analyze(ctx context.Context, req AnalyzeRouteRequest) (AnalyzeRouteResponse, error) {
	return AnalyzeRouteResponse{
		Route: RouteSummary{
			DistanceMeters:  5421,
			DurationSeconds: 812,
		},
		Analysis: RouteAnalysis{
			Difficulty: 4.7,
			Categories: RouteCategoryScores{
				Hills:         6.2,
				Curves:        4.1,
				Intersections: 3.8,
				HighSpeed:     2.0,
			},
		},
		Events: []any{},
	}, nil
}
