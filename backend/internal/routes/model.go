package routes

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
	DistanceMeters  int    `json:"distanceMeters"`
	DurationSeconds int    `json:"durationSeconds"`
	Polyline        string `json:"polyline"`
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
