package routes

import "testing"

func TestAnalyzeRoadEvents(t *testing.T) {
	events := AnalyzeRoadEvents([]RoadEventHint{
		{
			SegmentSequence:     0,
			Position:            Coordinate{Latitude: -26.9, Longitude: -49.1},
			RouteDistanceMeters: 120,
			NodeType:            "street_intersection",
			IntersectingEdges:   2,
		},
		{
			SegmentSequence:      1,
			Position:             Coordinate{Latitude: -26.91, Longitude: -49.11},
			RouteDistanceMeters:  220,
			NodeType:             "street_intersection",
			IntersectingEdges:    4,
			InternalIntersection: true,
		},
		{
			SegmentSequence:     2,
			Position:            Coordinate{Latitude: -26.92, Longitude: -49.12},
			RouteDistanceMeters: 320,
			RoadClass:           "residential",
			RoadUse:             "road",
			Roundabout:          true,
			StopSign:            true,
			TrafficSignal:       true,
		},
		{
			SegmentSequence:     3,
			Position:            Coordinate{Latitude: -26.93, Longitude: -49.13},
			RouteDistanceMeters: 420,
			RoadClass:           "motorway",
			RoadUse:             "ramp",
			EnteringHighway:     true,
			ExitingHighway:      true,
		},
	})

	expectedTypes := []string{
		"INTERSECTION",
		"COMPLEX_INTERSECTION",
		"STOP",
		"TRAFFIC_LIGHT",
		"ROUNDABOUT",
		"HIGHWAY_ENTRY",
		"HIGHWAY_EXIT",
	}
	if len(events) != len(expectedTypes) {
		t.Fatalf("expected %d events, got %d", len(expectedTypes), len(events))
	}
	for i, expected := range expectedTypes {
		if events[i].Type != expected {
			t.Fatalf("expected event %d to be %s, got %s", i, expected, events[i].Type)
		}
	}
}

func TestAnalyzeRoadEventsSkipsWeakIntersections(t *testing.T) {
	events := AnalyzeRoadEvents([]RoadEventHint{
		{
			SegmentSequence:     0,
			Position:            Coordinate{Latitude: -26.9, Longitude: -49.1},
			RouteDistanceMeters: 120,
			NodeType:            "street_intersection",
			IntersectingEdges:   1,
		},
	})

	if len(events) != 0 {
		t.Fatalf("expected no events, got %d", len(events))
	}
}

func TestAnalyzeSegmentRoadEventsDetectsRoadTransitions(t *testing.T) {
	events := AnalyzeSegmentRoadEvents([]RouteSegmentResult{
		{
			Sequence:       0,
			Geometry:       []Coordinate{{Latitude: 0, Longitude: 0}, {Latitude: 0, Longitude: 0.001}},
			DistanceMeters: 100,
			RoadClass:      "tertiary",
			RoadName:       "Rua A",
			RoadUse:        "road",
		},
		{
			Sequence:       1,
			Geometry:       []Coordinate{{Latitude: 0, Longitude: 0.001}, {Latitude: 0.001, Longitude: 0.001}},
			DistanceMeters: 100,
			RoadClass:      "tertiary",
			RoadName:       "Rua B",
			RoadUse:        "road",
		},
		{
			Sequence:       2,
			Geometry:       []Coordinate{{Latitude: 0.001, Longitude: 0.001}, {Latitude: 0.002, Longitude: 0.001}},
			DistanceMeters: 100,
			RoadClass:      "primary",
			RoadName:       "Avenida C",
			RoadUse:        "road",
		},
	})

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Type != "INTERSECTION" {
		t.Fatalf("expected first event INTERSECTION, got %q", events[0].Type)
	}
	if events[1].Type != "COMPLEX_INTERSECTION" {
		t.Fatalf("expected second event COMPLEX_INTERSECTION, got %q", events[1].Type)
	}
	if events[0].RouteDistanceMeters != 100 {
		t.Fatalf("expected first event at 100m, got %d", events[0].RouteDistanceMeters)
	}
}

func TestAnalyzeSegmentRoadEventsDoesNotMakeSameClassPrimaryTransitionsComplex(t *testing.T) {
	events := AnalyzeSegmentRoadEvents([]RouteSegmentResult{
		{
			Sequence:       0,
			Geometry:       []Coordinate{{Latitude: 0, Longitude: 0}, {Latitude: 0, Longitude: 0.001}},
			DistanceMeters: 100,
			RoadClass:      "primary",
			RoadName:       "Rua A",
			RoadUse:        "road",
		},
		{
			Sequence:       1,
			Geometry:       []Coordinate{{Latitude: 0, Longitude: 0.001}, {Latitude: 0.001, Longitude: 0.001}},
			DistanceMeters: 100,
			RoadClass:      "primary",
			RoadName:       "Rua B",
			RoadUse:        "road",
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "INTERSECTION" {
		t.Fatalf("expected INTERSECTION event, got %q", events[0].Type)
	}
}

func TestAnalyzeRoadEventsDedupesRoundaboutEdges(t *testing.T) {
	events := AnalyzeRoadEvents([]RoadEventHint{
		{
			SegmentSequence:     0,
			Position:            Coordinate{Latitude: -26.8, Longitude: -49.1},
			RouteDistanceMeters: 552,
			Roundabout:          true,
			RoadClass:           "primary",
			RoadUse:             "road",
		},
		{
			SegmentSequence:     0,
			Position:            Coordinate{Latitude: -26.8, Longitude: -49.1},
			RouteDistanceMeters: 565,
			Roundabout:          true,
			RoadClass:           "primary",
			RoadUse:             "road",
		},
		{
			SegmentSequence:     0,
			Position:            Coordinate{Latitude: -26.8, Longitude: -49.1},
			RouteDistanceMeters: 575,
			Roundabout:          true,
			RoadClass:           "primary",
			RoadUse:             "road",
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 deduped roundabout event, got %d", len(events))
	}
	if events[0].Type != "ROUNDABOUT" {
		t.Fatalf("expected ROUNDABOUT event, got %q", events[0].Type)
	}
}

func TestRoadEventDedupeCollapsesNearbyHighwayTransitions(t *testing.T) {
	events := dedupeRoadEvents([]RouteEventResult{
		{
			Type:                "HIGHWAY_EXIT",
			RouteDistanceMeters: 4189,
			Position:            Coordinate{Latitude: -26.8, Longitude: -49.1},
		},
		{
			Type:                "HIGHWAY_EXIT",
			RouteDistanceMeters: 4440,
			Position:            Coordinate{Latitude: -26.8, Longitude: -49.1},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 deduped highway exit event, got %d", len(events))
	}
}
