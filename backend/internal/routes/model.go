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
	Route    RouteSummary         `json:"route"`
	Analysis RouteAnalysis        `json:"analysis"`
	Events   []RouteEventResponse `json:"events"`
}

type RouteSummary struct {
	ID              string                 `json:"id,omitempty"`
	DistanceMeters  int                    `json:"distanceMeters"`
	DurationSeconds int                    `json:"durationSeconds"`
	Polyline        string                 `json:"polyline"`
	Segments        []RouteSegmentResponse `json:"segments,omitempty"`
}

type Route struct {
	ID              string
	Source          string
	Geometry        []Coordinate
	DistanceMeters  int
	DurationSeconds int
	Polyline        string
	Segments        []RouteSegment
	Events          []RouteEvent
	CreatedAt       time.Time
}

type RouteSegment struct {
	ID              string
	RouteID         string
	Sequence        int
	Geometry        []Coordinate
	DistanceMeters  int
	DurationSeconds int
	RoadClass       string
	RoadName        string
	RoadUse         string
	SpeedLimitKph   int
	ElevationStartM *float64
	ElevationEndM   *float64
	InclineAvgPct   *float64
	InclineMaxPct   *float64
	CreatedAt       time.Time
}

type RouteSegmentResult struct {
	Sequence        int
	Geometry        []Coordinate
	DistanceMeters  int
	DurationSeconds int
	RoadClass       string
	RoadName        string
	RoadUse         string
	SpeedLimitKph   int
	ElevationStartM *float64
	ElevationEndM   *float64
	InclineAvgPct   *float64
	InclineMaxPct   *float64
}

type ElevationSample struct {
	RouteDistanceMeters float64
	ElevationMeters     float64
}

type RouteEvent struct {
	ID                  string
	RouteID             string
	SegmentID           string
	Type                string
	Position            Coordinate
	RouteDistanceMeters int
	DifficultyScore     float64
	Metadata            map[string]any
	CreatedAt           time.Time
}

type RouteEventResult struct {
	SegmentSequence     int
	Type                string
	Position            Coordinate
	RouteDistanceMeters int
	DifficultyScore     float64
	Metadata            map[string]any
}

type GetRouteResponse struct {
	Route  PersistedRouteResponse `json:"route"`
	Events []RouteEventResponse   `json:"events"`
}

type PersistedRouteResponse struct {
	ID              string                 `json:"id"`
	Source          string                 `json:"source"`
	Geometry        []Coordinate           `json:"geometry"`
	DistanceMeters  int                    `json:"distanceMeters"`
	DurationSeconds int                    `json:"durationSeconds"`
	Polyline        string                 `json:"polyline"`
	Segments        []RouteSegmentResponse `json:"segments"`
	CreatedAt       time.Time              `json:"createdAt"`
}

type RouteSegmentResponse struct {
	ID              string       `json:"id,omitempty"`
	Sequence        int          `json:"sequence"`
	Geometry        []Coordinate `json:"geometry"`
	DistanceMeters  int          `json:"distanceMeters"`
	DurationSeconds int          `json:"durationSeconds,omitempty"`
	RoadClass       string       `json:"roadClass,omitempty"`
	RoadName        string       `json:"roadName,omitempty"`
	RoadUse         string       `json:"roadUse,omitempty"`
	SpeedLimitKph   int          `json:"speedLimitKph,omitempty"`
	ElevationStartM *float64     `json:"elevationStartMeters,omitempty"`
	ElevationEndM   *float64     `json:"elevationEndMeters,omitempty"`
	InclineAvgPct   *float64     `json:"inclineAvgPercent,omitempty"`
	InclineMaxPct   *float64     `json:"inclineMaxPercent,omitempty"`
}

type RouteEventResponse struct {
	ID                  string         `json:"id,omitempty"`
	SegmentID           string         `json:"segmentId,omitempty"`
	Type                string         `json:"type"`
	Position            Coordinate     `json:"position"`
	RouteDistanceMeters int            `json:"routeDistanceMeters,omitempty"`
	DifficultyScore     float64        `json:"difficultyScore,omitempty"`
	Metadata            map[string]any `json:"metadata,omitempty"`
	CreatedAt           time.Time      `json:"createdAt,omitempty"`
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
