package routes

import "testing"

func TestAnalyzeCompoundEventsDetectsHillStop(t *testing.T) {
	events := AnalyzeCompoundEvents([]RouteEventResult{
		{
			SegmentSequence:     0,
			Type:                "STEEP_HILL",
			Position:            Coordinate{Latitude: 0, Longitude: 0},
			RouteDistanceMeters: 300,
			DifficultyScore:     7.2,
			Metadata: map[string]any{
				"direction":         "uphill",
				"inclineMaxPercent": 6.0,
			},
		},
		{
			SegmentSequence:     1,
			Type:                "STOP",
			Position:            Coordinate{Latitude: 0.0001, Longitude: 0.0001},
			RouteDistanceMeters: 335,
			DifficultyScore:     3.5,
			Metadata:            map[string]any{},
		},
	}, 1000)

	if len(events) != 1 {
		t.Fatalf("expected 1 compound event, got %d", len(events))
	}
	if events[0].Type != "HILL_STOP" {
		t.Fatalf("expected HILL_STOP, got %q", events[0].Type)
	}
	if events[0].RouteDistanceMeters != 335 {
		t.Fatalf("expected compound event at stop distance 335, got %d", events[0].RouteDistanceMeters)
	}
	if events[0].DifficultyScore != 8.7 {
		t.Fatalf("expected difficulty score 8.7, got %v", events[0].DifficultyScore)
	}
	if events[0].Metadata["associationDistanceMeters"] != 35 {
		t.Fatalf("expected association distance 35, got %v", events[0].Metadata["associationDistanceMeters"])
	}
}

func TestAnalyzeCompoundEventsSkipsDistantHillStop(t *testing.T) {
	events := AnalyzeCompoundEvents([]RouteEventResult{
		{
			Type:                "HILL",
			RouteDistanceMeters: 100,
			DifficultyScore:     4,
			Metadata:            map[string]any{},
		},
		{
			Type:                "STOP",
			RouteDistanceMeters: 220,
			DifficultyScore:     3.5,
			Metadata:            map[string]any{},
		},
	}, 1000)

	if len(events) != 0 {
		t.Fatalf("expected no compound events, got %d", len(events))
	}
}

func TestAnalyzeCompoundEventsDetectsControlledComplexIntersection(t *testing.T) {
	events := AnalyzeCompoundEvents([]RouteEventResult{
		{
			SegmentSequence:     2,
			Type:                "INTERSECTION",
			Position:            Coordinate{Latitude: 0, Longitude: 0},
			RouteDistanceMeters: 500,
			DifficultyScore:     3,
			Metadata:            map[string]any{},
		},
		{
			SegmentSequence:     2,
			Type:                "TRAFFIC_LIGHT",
			Position:            Coordinate{Latitude: 0, Longitude: 0},
			RouteDistanceMeters: 515,
			DifficultyScore:     4,
			Metadata:            map[string]any{},
		},
	}, 1000)

	if len(events) != 1 {
		t.Fatalf("expected 1 compound event, got %d", len(events))
	}
	if events[0].Type != "COMPLEX_INTERSECTION" {
		t.Fatalf("expected COMPLEX_INTERSECTION, got %q", events[0].Type)
	}
	if events[0].RouteDistanceMeters != 500 {
		t.Fatalf("expected complex intersection at 500m, got %d", events[0].RouteDistanceMeters)
	}
}

func TestAnalyzeCompoundEventsDoesNotDuplicateExistingComplexIntersection(t *testing.T) {
	events := AnalyzeCompoundEvents([]RouteEventResult{
		{
			Type:                "INTERSECTION",
			RouteDistanceMeters: 500,
			DifficultyScore:     3,
			Metadata:            map[string]any{},
		},
		{
			Type:                "COMPLEX_INTERSECTION",
			RouteDistanceMeters: 505,
			DifficultyScore:     6,
			Metadata:            map[string]any{},
		},
		{
			Type:                "STOP",
			RouteDistanceMeters: 510,
			DifficultyScore:     3.5,
			Metadata:            map[string]any{},
		},
	}, 1000)

	if len(events) != 0 {
		t.Fatalf("expected no additional compound events, got %d", len(events))
	}
}

func TestAnalyzeCompoundEventsDetectsRouteStartHillStop(t *testing.T) {
	events := AnalyzeCompoundEvents([]RouteEventResult{
		{
			SegmentSequence:     0,
			Type:                "STEEP_HILL",
			Position:            Coordinate{Latitude: 0, Longitude: 0},
			RouteDistanceMeters: 30,
			DifficultyScore:     10,
			Metadata: map[string]any{
				"direction":         "uphill",
				"inclineMaxPercent": 11.2,
			},
		},
		{
			Type:                "INTERSECTION",
			RouteDistanceMeters: 300,
			DifficultyScore:     3,
			Metadata:            map[string]any{},
		},
	}, 1000)

	if len(events) != 1 {
		t.Fatalf("expected 1 compound event, got %d", len(events))
	}
	if events[0].Type != "HILL_STOP" {
		t.Fatalf("expected HILL_STOP, got %q", events[0].Type)
	}
	if events[0].RouteDistanceMeters != 0 {
		t.Fatalf("expected route start stop at 0m, got %d", events[0].RouteDistanceMeters)
	}
	if events[0].Metadata["stopContext"] != "route_start" {
		t.Fatalf("expected route_start stop context, got %v", events[0].Metadata["stopContext"])
	}
}

func TestAnalyzeCompoundEventsDetectsRouteEndHillStop(t *testing.T) {
	events := AnalyzeCompoundEvents([]RouteEventResult{
		{
			Type:                "INTERSECTION",
			RouteDistanceMeters: 300,
			DifficultyScore:     3,
			Metadata:            map[string]any{},
		},
		{
			SegmentSequence:     4,
			Type:                "HILL",
			Position:            Coordinate{Latitude: 0.1, Longitude: 0.1},
			RouteDistanceMeters: 960,
			DifficultyScore:     5.5,
			Metadata: map[string]any{
				"direction":         "downhill",
				"inclineMaxPercent": -4.6,
			},
		},
	}, 1000)

	if len(events) != 1 {
		t.Fatalf("expected 1 compound event, got %d", len(events))
	}
	if events[0].RouteDistanceMeters != 1000 {
		t.Fatalf("expected route end stop at 1000m, got %d", events[0].RouteDistanceMeters)
	}
	if events[0].Metadata["stopContext"] != "route_end" {
		t.Fatalf("expected route_end stop context, got %v", events[0].Metadata["stopContext"])
	}
}
