package routes

import "math"

const (
	minCurvePointTurnDegrees   = 4.0
	minCurveTotalTurnDegrees   = 25.0
	sharpCurveTotalTurnDegrees = 60.0
	minCurveDistanceMeters     = 15.0
	maxCurveSequenceGapMeters  = 150
	minCurveSequenceEventCount = 3
	earthRadiusMeters          = 6371000.0
	curveBoundaryPointGrace    = 1
	curveDirectionLeft         = "left"
	curveDirectionRight        = "right"
)

type curveCandidate struct {
	segmentSequence int
	startIndex      int
	endIndex        int
	startMeters     float64
	endMeters       float64
	totalTurn       float64
	maxPointTurn    float64
	pointTurns      int
	direction       string
	position        Coordinate
}

// AnalyzeCurves detects driving-relevant continuous curves from route segment geometry.
func AnalyzeCurves(segments []RouteSegmentResult) []RouteEventResult {
	events := []RouteEventResult{}
	routeDistanceStart := 0.0

	for _, segment := range segments {
		candidates := curveCandidates(segment, routeDistanceStart)
		for _, candidate := range candidates {
			eventType := "CURVE"
			if candidate.totalTurn >= sharpCurveTotalTurnDegrees {
				eventType = "SHARP_CURVE"
			}

			events = append(events, RouteEventResult{
				SegmentSequence:     candidate.segmentSequence,
				Type:                eventType,
				Position:            candidate.position,
				RouteDistanceMeters: int(math.Round((candidate.startMeters + candidate.endMeters) / 2)),
				DifficultyScore:     round(math.Min(candidate.totalTurn/12, 10), 2),
				Metadata: map[string]any{
					"direction":        candidate.direction,
					"totalTurnDegrees": round(candidate.totalTurn, 2),
					"maxPointTurnDeg":  round(candidate.maxPointTurn, 2),
					"lengthMeters":     int(math.Round(candidate.endMeters - candidate.startMeters)),
					"pointTurns":       candidate.pointTurns,
				},
			})
		}

		routeDistanceStart += float64(segment.DistanceMeters)
	}

	return append(events, curveSequenceEvents(events)...)
}

func curveCandidates(segment RouteSegmentResult, routeDistanceStart float64) []curveCandidate {
	if len(segment.Geometry) < 3 || segment.DistanceMeters <= 0 {
		return nil
	}

	distances := cumulativeGeometryDistances(segment.Geometry)
	totalGeometryDistance := distances[len(distances)-1]
	if totalGeometryDistance <= 0 {
		return nil
	}

	candidates := []curveCandidate{}
	var active *curveCandidate

	for pointIndex := 1; pointIndex < len(segment.Geometry)-1; pointIndex++ {
		turn := signedHeadingChange(segment.Geometry[pointIndex-1], segment.Geometry[pointIndex], segment.Geometry[pointIndex+1])
		absoluteTurn := math.Abs(turn)
		if absoluteTurn < minCurvePointTurnDegrees {
			if active != nil {
				candidates = finishCurveCandidate(candidates, *active, distances, segment, routeDistanceStart)
				active = nil
			}
			continue
		}

		direction := curveDirectionRight
		if turn < 0 {
			direction = curveDirectionLeft
		}

		if active == nil || active.direction != direction {
			if active != nil {
				candidates = finishCurveCandidate(candidates, *active, distances, segment, routeDistanceStart)
			}
			active = &curveCandidate{
				segmentSequence: segment.Sequence,
				startIndex:      pointIndex - 1,
				endIndex:        pointIndex + 1,
				direction:       direction,
			}
		}

		active.endIndex = pointIndex + 1
		active.totalTurn += absoluteTurn
		active.maxPointTurn = math.Max(active.maxPointTurn, absoluteTurn)
		active.pointTurns++
	}

	if active != nil {
		candidates = finishCurveCandidate(candidates, *active, distances, segment, routeDistanceStart)
	}

	return candidates
}

func finishCurveCandidate(candidates []curveCandidate, candidate curveCandidate, distances []float64, segment RouteSegmentResult, routeDistanceStart float64) []curveCandidate {
	if candidate.pointTurns == 1 &&
		(candidate.startIndex <= curveBoundaryPointGrace ||
			candidate.endIndex >= len(segment.Geometry)-1-curveBoundaryPointGrace) {
		return candidates
	}

	totalGeometryDistance := distances[len(distances)-1]
	startMeters := routeDistanceStart + scaledRouteDistance(distances[candidate.startIndex], totalGeometryDistance, segment.DistanceMeters)
	endMeters := routeDistanceStart + scaledRouteDistance(distances[candidate.endIndex], totalGeometryDistance, segment.DistanceMeters)
	if endMeters-startMeters < minCurveDistanceMeters || candidate.totalTurn < minCurveTotalTurnDegrees {
		return candidates
	}

	candidate.startMeters = startMeters
	candidate.endMeters = endMeters
	candidate.position = coordinateAtDistance(segment.Geometry, distances, (distances[candidate.startIndex]+distances[candidate.endIndex])/2)
	return append(candidates, candidate)
}

func curveSequenceEvents(events []RouteEventResult) []RouteEventResult {
	sequences := []RouteEventResult{}
	start := -1

	for i, event := range events {
		if event.Type != "CURVE" && event.Type != "SHARP_CURVE" {
			continue
		}
		if start == -1 {
			start = i
			continue
		}
		if event.RouteDistanceMeters-events[i-1].RouteDistanceMeters > maxCurveSequenceGapMeters {
			sequences = appendCurveSequence(sequences, events[start:i])
			start = i
		}
	}
	if start != -1 {
		sequences = appendCurveSequence(sequences, events[start:])
	}

	return sequences
}

func appendCurveSequence(sequences []RouteEventResult, events []RouteEventResult) []RouteEventResult {
	if len(events) < minCurveSequenceEventCount {
		return sequences
	}

	totalTurn := 0.0
	for _, event := range events {
		if value, ok := event.Metadata["totalTurnDegrees"].(float64); ok {
			totalTurn += value
		}
	}

	mid := events[len(events)/2]
	return append(sequences, RouteEventResult{
		SegmentSequence:     mid.SegmentSequence,
		Type:                "CURVE_SEQUENCE",
		Position:            mid.Position,
		RouteDistanceMeters: mid.RouteDistanceMeters,
		DifficultyScore:     round(math.Min(float64(len(events))*1.5+totalTurn/90, 10), 2),
		Metadata: map[string]any{
			"curveCount":       len(events),
			"totalTurnDegrees": round(totalTurn, 2),
		},
	})
}

func signedHeadingChange(a Coordinate, b Coordinate, c Coordinate) float64 {
	first := bearingDegrees(a, b)
	second := bearingDegrees(b, c)
	delta := second - first
	for delta > 180 {
		delta -= 360
	}
	for delta < -180 {
		delta += 360
	}
	return delta
}

func bearingDegrees(a Coordinate, b Coordinate) float64 {
	lat1 := degreesToRadians(a.Latitude)
	lat2 := degreesToRadians(b.Latitude)
	deltaLon := degreesToRadians(b.Longitude - a.Longitude)

	y := math.Sin(deltaLon) * math.Cos(lat2)
	x := math.Cos(lat1)*math.Sin(lat2) - math.Sin(lat1)*math.Cos(lat2)*math.Cos(deltaLon)
	bearing := math.Atan2(y, x) * 180 / math.Pi
	if bearing < 0 {
		bearing += 360
	}
	return bearing
}

func cumulativeGeometryDistances(coordinates []Coordinate) []float64 {
	distances := make([]float64, len(coordinates))
	for i := 1; i < len(coordinates); i++ {
		distances[i] = distances[i-1] + haversineMeters(coordinates[i-1], coordinates[i])
	}
	return distances
}

func scaledRouteDistance(geometryDistance float64, totalGeometryDistance float64, segmentDistanceMeters int) float64 {
	return (geometryDistance / totalGeometryDistance) * float64(segmentDistanceMeters)
}

func coordinateAtDistance(coordinates []Coordinate, distances []float64, targetMeters float64) Coordinate {
	for i := 1; i < len(coordinates); i++ {
		if distances[i] < targetMeters {
			continue
		}

		interval := distances[i] - distances[i-1]
		if interval <= 0 {
			return coordinates[i]
		}

		ratio := (targetMeters - distances[i-1]) / interval
		return Coordinate{
			Latitude:  coordinates[i-1].Latitude + (coordinates[i].Latitude-coordinates[i-1].Latitude)*ratio,
			Longitude: coordinates[i-1].Longitude + (coordinates[i].Longitude-coordinates[i-1].Longitude)*ratio,
		}
	}
	return coordinates[len(coordinates)-1]
}

func haversineMeters(a Coordinate, b Coordinate) float64 {
	lat1 := degreesToRadians(a.Latitude)
	lat2 := degreesToRadians(b.Latitude)
	deltaLat := degreesToRadians(b.Latitude - a.Latitude)
	deltaLon := degreesToRadians(b.Longitude - a.Longitude)

	h := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
}

func degreesToRadians(value float64) float64 {
	return value * math.Pi / 180
}
