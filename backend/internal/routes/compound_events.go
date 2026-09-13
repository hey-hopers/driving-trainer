package routes

import "math"

const (
	hillStopAssociationDistanceMeters            = 75
	complexIntersectionAssociationDistanceMeters = 50
)

func AnalyzeCompoundEvents(events []RouteEventResult, routeDistanceMeters int) []RouteEventResult {
	if len(events) == 0 {
		return nil
	}

	compounds := []RouteEventResult{}
	compounds = append(compounds, hillStopEvents(events)...)
	compounds = append(compounds, routeBoundaryHillStopEvents(events, routeDistanceMeters)...)
	compounds = append(compounds, compoundComplexIntersectionEvents(events)...)

	return dedupeCompoundEvents(compounds)
}

func hillStopEvents(events []RouteEventResult) []RouteEventResult {
	compounds := []RouteEventResult{}
	for _, hill := range events {
		if !isHillEvent(hill.Type) {
			continue
		}
		for _, stop := range events {
			if stop.Type != "STOP" {
				continue
			}
			if eventDistanceMeters(hill, stop) > hillStopAssociationDistanceMeters {
				continue
			}

			compounds = append(compounds, RouteEventResult{
				SegmentSequence:     stop.SegmentSequence,
				Type:                "HILL_STOP",
				Position:            stop.Position,
				RouteDistanceMeters: stop.RouteDistanceMeters,
				DifficultyScore:     compoundScore(hill, stop, 1.5),
				Metadata: map[string]any{
					"source":                    "compound_event",
					"components":                []string{hill.Type, stop.Type},
					"associationDistanceMeters": eventDistanceMeters(hill, stop),
					"hillRouteDistanceMeters":   hill.RouteDistanceMeters,
					"stopRouteDistanceMeters":   stop.RouteDistanceMeters,
					"hillDirection":             hill.Metadata["direction"],
					"hillInclineMaxPercent":     hill.Metadata["inclineMaxPercent"],
				},
			})
		}
	}
	return compounds
}

func routeBoundaryHillStopEvents(events []RouteEventResult, routeDistanceMeters int) []RouteEventResult {
	if routeDistanceMeters <= 0 {
		return nil
	}

	compounds := []RouteEventResult{}
	for _, hill := range events {
		if !isHillEvent(hill.Type) {
			continue
		}

		if hill.RouteDistanceMeters <= hillStopAssociationDistanceMeters {
			compounds = append(compounds, routeBoundaryHillStopEvent(hill, "route_start", 0))
		}

		distanceFromEnd := routeDistanceMeters - hill.RouteDistanceMeters
		if distanceFromEnd >= 0 && distanceFromEnd <= hillStopAssociationDistanceMeters {
			compounds = append(compounds, routeBoundaryHillStopEvent(hill, "route_end", routeDistanceMeters))
		}
	}
	return compounds
}

func routeBoundaryHillStopEvent(hill RouteEventResult, stopContext string, stopDistanceMeters int) RouteEventResult {
	return RouteEventResult{
		SegmentSequence:     hill.SegmentSequence,
		Type:                "HILL_STOP",
		Position:            hill.Position,
		RouteDistanceMeters: stopDistanceMeters,
		DifficultyScore:     compoundScore(hill, RouteEventResult{}, 1),
		Metadata: map[string]any{
			"source":                    "compound_event",
			"components":                []string{hill.Type, "ROUTE_BOUNDARY_STOP"},
			"stopContext":               stopContext,
			"associationDistanceMeters": eventDistanceMeters(hill, RouteEventResult{RouteDistanceMeters: stopDistanceMeters}),
			"hillRouteDistanceMeters":   hill.RouteDistanceMeters,
			"stopRouteDistanceMeters":   stopDistanceMeters,
			"hillDirection":             hill.Metadata["direction"],
			"hillInclineMaxPercent":     hill.Metadata["inclineMaxPercent"],
		},
	}
}

func compoundComplexIntersectionEvents(events []RouteEventResult) []RouteEventResult {
	compounds := []RouteEventResult{}
	for _, intersection := range events {
		if intersection.Type != "INTERSECTION" {
			continue
		}
		if hasNearbyEvent(events, "COMPLEX_INTERSECTION", intersection.RouteDistanceMeters, complexIntersectionAssociationDistanceMeters) {
			continue
		}

		control, ok := nearestEventOfTypes(events, intersection, []string{"STOP", "TRAFFIC_LIGHT", "ROUNDABOUT"})
		if !ok {
			continue
		}

		compounds = append(compounds, RouteEventResult{
			SegmentSequence:     intersection.SegmentSequence,
			Type:                "COMPLEX_INTERSECTION",
			Position:            intersection.Position,
			RouteDistanceMeters: intersection.RouteDistanceMeters,
			DifficultyScore:     compoundScore(intersection, control, 1),
			Metadata: map[string]any{
				"source":                     "compound_event",
				"components":                 []string{intersection.Type, control.Type},
				"associationDistanceMeters":  eventDistanceMeters(intersection, control),
				"intersectionDistanceMeters": intersection.RouteDistanceMeters,
				"controlDistanceMeters":      control.RouteDistanceMeters,
			},
		})
	}
	return compounds
}

func isHillEvent(eventType string) bool {
	return eventType == "HILL" || eventType == "STEEP_HILL"
}

func nearestEventOfTypes(events []RouteEventResult, origin RouteEventResult, eventTypes []string) (RouteEventResult, bool) {
	var nearest RouteEventResult
	nearestDistance := math.MaxInt

	for _, candidate := range events {
		if !containsEventType(eventTypes, candidate.Type) {
			continue
		}
		distance := eventDistanceMeters(origin, candidate)
		if distance > complexIntersectionAssociationDistanceMeters || distance >= nearestDistance {
			continue
		}
		nearest = candidate
		nearestDistance = distance
	}

	return nearest, nearestDistance != math.MaxInt
}

func containsEventType(eventTypes []string, eventType string) bool {
	for _, candidate := range eventTypes {
		if candidate == eventType {
			return true
		}
	}
	return false
}

func hasNearbyEvent(events []RouteEventResult, eventType string, routeDistanceMeters int, thresholdMeters int) bool {
	for _, event := range events {
		if event.Type == eventType && int(math.Abs(float64(event.RouteDistanceMeters-routeDistanceMeters))) <= thresholdMeters {
			return true
		}
	}
	return false
}

func eventDistanceMeters(first RouteEventResult, second RouteEventResult) int {
	return int(math.Abs(float64(first.RouteDistanceMeters - second.RouteDistanceMeters)))
}

func compoundScore(first RouteEventResult, second RouteEventResult, bonus float64) float64 {
	return round(math.Min(math.Max(first.DifficultyScore, second.DifficultyScore)+bonus, 10), 2)
}

func dedupeCompoundEvents(events []RouteEventResult) []RouteEventResult {
	deduped := []RouteEventResult{}
	for _, event := range events {
		duplicate := false
		for _, existing := range deduped {
			if existing.Type == event.Type &&
				eventDistanceMeters(existing, event) <= defaultRoadEventDedupeDistanceMeters {
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
