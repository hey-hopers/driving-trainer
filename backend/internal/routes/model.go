package routes

import "time"

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type AnalyzeRouteRequest struct {
	Origin      *Coordinate `json:"origin"`
	Destination *Coordinate `json:"destination"`
}

type AnalyzeRouteResponse struct {
	Route    RouteSummary  `json:"route"`
	Analysis RouteAnalysis `json:"analysis"`
	Events   []any         `json:"events"`
}

type RouteSummary struct {
	ID              string `json:"id,omitempty"`
	DistanceMeters  int    `json:"distanceMeters"`
	DurationSeconds int    `json:"durationSeconds"`
	Polyline        string `json:"polyline"`
}

type Route struct {
	ID              string
	Source          string
	Geometry        []Coordinate
	DistanceMeters  int
	DurationSeconds int
	Polyline        string
	CreatedAt       time.Time
}

type GetRouteResponse struct {
	Route PersistedRouteResponse `json:"route"`
}

type PersistedRouteResponse struct {
	ID              string       `json:"id"`
	Source          string       `json:"source"`
	Geometry        []Coordinate `json:"geometry"`
	DistanceMeters  int          `json:"distanceMeters"`
	DurationSeconds int          `json:"durationSeconds"`
	Polyline        string       `json:"polyline"`
	CreatedAt       time.Time    `json:"createdAt"`
}

type RouteAnalysis struct {
	Difficulty float64             `json:"difficulty"`
	Categories RouteCategoryScores `json:"categories"`
}

type RouteCategoryScores struct {
	Hills         float64 `json:"hills"`
	Curves        float64 `json:"curves"`
	Intersections float64 `json:"intersections"`
	HighSpeed     float64 `json:"highSpeed"`
}
