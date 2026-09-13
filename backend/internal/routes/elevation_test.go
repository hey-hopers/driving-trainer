package routes

import "testing"

func TestAnalyzeElevationDetectsSteepHill(t *testing.T) {
	segments := []RouteSegmentResult{
		{
			Sequence:       0,
			Geometry:       []Coordinate{{Latitude: 0, Longitude: 0}, {Latitude: 0, Longitude: 0.001}},
			DistanceMeters: 100,
		},
	}
	profile := []ElevationSample{
		{RouteDistanceMeters: 0, ElevationMeters: 10},
		{RouteDistanceMeters: 50, ElevationMeters: 12},
		{RouteDistanceMeters: 100, ElevationMeters: 18},
	}

	enriched, events := AnalyzeElevation(segments, profile)

	if len(enriched) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(enriched))
	}
	if enriched[0].ElevationStartM == nil || *enriched[0].ElevationStartM != 10 {
		t.Fatalf("expected elevation start 10, got %v", enriched[0].ElevationStartM)
	}
	if enriched[0].ElevationEndM == nil || *enriched[0].ElevationEndM != 18 {
		t.Fatalf("expected elevation end 18, got %v", enriched[0].ElevationEndM)
	}
	if enriched[0].InclineAvgPct == nil || *enriched[0].InclineAvgPct != 8 {
		t.Fatalf("expected average incline 8, got %v", enriched[0].InclineAvgPct)
	}
	if enriched[0].InclineMaxPct == nil || *enriched[0].InclineMaxPct != 12 {
		t.Fatalf("expected max incline 12, got %v", enriched[0].InclineMaxPct)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "STEEP_HILL" {
		t.Fatalf("expected STEEP_HILL event, got %q", events[0].Type)
	}
	if events[0].Metadata["direction"] != "uphill" {
		t.Fatalf("expected uphill direction, got %v", events[0].Metadata["direction"])
	}
}

func TestAnalyzeElevationDetectsDownhill(t *testing.T) {
	segments := []RouteSegmentResult{
		{
			Sequence:       0,
			Geometry:       []Coordinate{{Latitude: 0, Longitude: 0}, {Latitude: 0, Longitude: 0.001}},
			DistanceMeters: 100,
		},
	}
	profile := []ElevationSample{
		{RouteDistanceMeters: 0, ElevationMeters: 20},
		{RouteDistanceMeters: 100, ElevationMeters: 16},
	}

	_, events := AnalyzeElevation(segments, profile)

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "HILL" {
		t.Fatalf("expected HILL event, got %q", events[0].Type)
	}
	if events[0].Metadata["direction"] != "downhill" {
		t.Fatalf("expected downhill direction, got %v", events[0].Metadata["direction"])
	}
}

func TestAnalyzeElevationSkipsFlatSegments(t *testing.T) {
	segments := []RouteSegmentResult{
		{
			Sequence:       0,
			Geometry:       []Coordinate{{Latitude: 0, Longitude: 0}, {Latitude: 0, Longitude: 0.001}},
			DistanceMeters: 100,
		},
	}
	profile := []ElevationSample{
		{RouteDistanceMeters: 0, ElevationMeters: 10},
		{RouteDistanceMeters: 100, ElevationMeters: 11},
	}

	_, events := AnalyzeElevation(segments, profile)

	if len(events) != 0 {
		t.Fatalf("expected no events, got %d", len(events))
	}
}
