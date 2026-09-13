package routes

import "math"

const (
	intersectionMinIntersectingEdges        = 2
	complexIntersectionMinIntersectingEdges = 4
	roundaboutDedupeDistanceMeters          = 35
	highwayTransitionDedupeDistanceMeters   = 350
	defaultRoadEventDedupeDistanceMeters    = 5
)

func AnalyzeRoadEvents(hints []RoadEventHint) []RouteEventResult {
	events := []RouteEventResult{}
	for _, hint := range hints {
		events = append(events, roadEventsFromHint(hint)...)
	}
	return dedupeRoadEvents(events)
}

func AnalyzeSegmentRoadEvents(segments []RouteSegmentResult) []RouteEventResult {
	if len(segments) < 2 {
		return nil
	}

	events := []RouteEventResult{}
	routeDistanceMeters := segments[0].DistanceMeters
	for i := 1; i < len(segments); i++ {
		previous := segments[i-1]
		current := segments[i]

		if isMeaningfulRoadTransition(previous, current) {
			eventType := "INTERSECTION"
			score := 3.0
			if isComplexRoadTransition(previous, current) {
				eventType = "COMPLEX_INTERSECTION"
				score = 5.5
			}

			events = append(events, RouteEventResult{
				SegmentSequence:     current.Sequence,
				Type:                eventType,
				Position:            firstCoordinate(current.Geometry),
				RouteDistanceMeters: routeDistanceMeters,
				DifficultyScore:     score,
				Metadata: map[string]any{
					"source":            "segment_transition",
					"fromRoadName":      previous.RoadName,
					"toRoadName":        current.RoadName,
					"fromRoadClass":     previous.RoadClass,
					"toRoadClass":       current.RoadClass,
					"fromRoadUse":       previous.RoadUse,
					"toRoadUse":         current.RoadUse,
					"fromSpeedLimitKph": previous.SpeedLimitKph,
					"toSpeedLimitKph":   current.SpeedLimitKph,
				},
			})
		}

		if isHighwayEntryTransition(previous, current) {
			events = append(events, segmentTransitionEvent(previous, current, "HIGHWAY_ENTRY", routeDistanceMeters, 6))
		}
		if isHighwayExitTransition(previous, current) {
			events = append(events, segmentTransitionEvent(previous, current, "HIGHWAY_EXIT", routeDistanceMeters, 5.5))
		}

		routeDistanceMeters += current.DistanceMeters
	}

	return dedupeRoadEvents(events)
}

func roadEventsFromHint(hint RoadEventHint) []RouteEventResult {
	events := []RouteEventResult{}

	if hint.StopSign {
		events = append(events, newRoadEvent(hint, "STOP", 3.5, nil))
	}
	if hint.TrafficSignal {
		events = append(events, newRoadEvent(hint, "TRAFFIC_LIGHT", 4, nil))
	}
	if hint.Roundabout {
		events = append(events, newRoadEvent(hint, "ROUNDABOUT", 5, map[string]any{
			"roadClass": hint.RoadClass,
			"roadUse":   hint.RoadUse,
		}))
	}
	if hint.EnteringHighway {
		events = append(events, newRoadEvent(hint, "HIGHWAY_ENTRY", 6, map[string]any{
			"roadClass": hint.RoadClass,
			"roadUse":   hint.RoadUse,
		}))
	}
	if hint.ExitingHighway {
		events = append(events, newRoadEvent(hint, "HIGHWAY_EXIT", 5.5, map[string]any{
			"roadClass": hint.RoadClass,
			"roadUse":   hint.RoadUse,
		}))
	}

	if isComplexIntersection(hint) {
		events = append(events, newRoadEvent(hint, "COMPLEX_INTERSECTION", 6, intersectionMetadata(hint)))
		return events
	}
	if isIntersection(hint) {
		events = append(events, newRoadEvent(hint, "INTERSECTION", 3, intersectionMetadata(hint)))
	}

	return events
}

func isIntersection(hint RoadEventHint) bool {
	return hint.NodeType == "street_intersection" &&
		hint.IntersectingEdges >= intersectionMinIntersectingEdges
}

func isComplexIntersection(hint RoadEventHint) bool {
	return isIntersection(hint) &&
		(hint.InternalIntersection ||
			hint.Fork ||
			hint.IntersectingEdges >= complexIntersectionMinIntersectingEdges)
}

func isMeaningfulRoadTransition(previous RouteSegmentResult, current RouteSegmentResult) bool {
	if len(current.Geometry) == 0 {
		return false
	}
	if previous.RoadName == "" && current.RoadName == "" &&
		previous.RoadClass == current.RoadClass &&
		previous.RoadUse == current.RoadUse {
		return false
	}
	if previous.RoadName != "" && previous.RoadName == current.RoadName &&
		previous.RoadClass == current.RoadClass &&
		previous.RoadUse == current.RoadUse {
		return false
	}
	return previous.RoadName != current.RoadName ||
		previous.RoadClass != current.RoadClass ||
		previous.RoadUse != current.RoadUse
}

func isComplexRoadTransition(previous RouteSegmentResult, current RouteSegmentResult) bool {
	return isHighwaySegment(previous) ||
		isHighwaySegment(current) ||
		isPrimaryClassChange(previous.RoadClass, current.RoadClass) ||
		previous.RoadUse == "ramp" ||
		current.RoadUse == "ramp" ||
		previous.RoadUse == "turn_channel" ||
		current.RoadUse == "turn_channel"
}

func isHighwayEntryTransition(previous RouteSegmentResult, current RouteSegmentResult) bool {
	if isHighwaySegment(previous) {
		return false
	}
	return isHighwaySegment(current) ||
		(current.RoadUse == "ramp" && isMajorRoadClass(current.RoadClass))
}

func isHighwayExitTransition(previous RouteSegmentResult, current RouteSegmentResult) bool {
	return isHighwaySegment(previous) && !isHighwaySegment(current)
}

func isHighwaySegment(segment RouteSegmentResult) bool {
	return segment.RoadClass == "motorway" || segment.RoadClass == "trunk"
}

func isMajorRoadClass(roadClass string) bool {
	return roadClass == "motorway" ||
		roadClass == "trunk" ||
		roadClass == "primary"
}

func isPrimaryClassChange(previousRoadClass string, currentRoadClass string) bool {
	return previousRoadClass != currentRoadClass &&
		(previousRoadClass == "primary" || currentRoadClass == "primary")
}

func segmentTransitionEvent(previous RouteSegmentResult, current RouteSegmentResult, eventType string, routeDistanceMeters int, score float64) RouteEventResult {
	return RouteEventResult{
		SegmentSequence:     current.Sequence,
		Type:                eventType,
		Position:            firstCoordinate(current.Geometry),
		RouteDistanceMeters: routeDistanceMeters,
		DifficultyScore:     score,
		Metadata: map[string]any{
			"source":        "segment_transition",
			"fromRoadClass": previous.RoadClass,
			"toRoadClass":   current.RoadClass,
			"fromRoadUse":   previous.RoadUse,
			"toRoadUse":     current.RoadUse,
		},
	}
}

func firstCoordinate(coordinates []Coordinate) Coordinate {
	if len(coordinates) == 0 {
		return Coordinate{}
	}
	return coordinates[0]
}

func intersectionMetadata(hint RoadEventHint) map[string]any {
	return map[string]any{
		"nodeType":          hint.NodeType,
		"intersectingEdges": hint.IntersectingEdges,
		"internal":          hint.InternalIntersection,
		"fork":              hint.Fork,
	}
}

func newRoadEvent(hint RoadEventHint, eventType string, score float64, metadata map[string]any) RouteEventResult {
	if metadata == nil {
		metadata = map[string]any{}
	}

	return RouteEventResult{
		SegmentSequence:     hint.SegmentSequence,
		Type:                eventType,
		Position:            hint.Position,
		RouteDistanceMeters: hint.RouteDistanceMeters,
		DifficultyScore:     score,
		Metadata:            metadata,
	}
}

func dedupeRoadEvents(events []RouteEventResult) []RouteEventResult {
	deduped := []RouteEventResult{}
	for _, event := range events {
		duplicate := false
		for _, existing := range deduped {
			distanceThresholdMeters := defaultRoadEventDedupeDistanceMeters
			if event.Type == "ROUNDABOUT" {
				distanceThresholdMeters = roundaboutDedupeDistanceMeters
			}
			if event.Type == "HIGHWAY_ENTRY" || event.Type == "HIGHWAY_EXIT" {
				distanceThresholdMeters = highwayTransitionDedupeDistanceMeters
			}
			if existing.Type == event.Type &&
				math.Abs(float64(existing.RouteDistanceMeters-event.RouteDistanceMeters)) <= float64(distanceThresholdMeters) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			deduped = append(deduped, event)
		}
	}
	return deduped
}
