package routes

import "testing"

func TestAnalyzeCurvesDetectsCurve(t *testing.T) {
	events := AnalyzeCurves([]RouteSegmentResult{
		{
			Sequence:       0,
			DistanceMeters: 120,
			Geometry: []Coordinate{
				{Latitude: 0, Longitude: 0},
				{Latitude: 0.0003, Longitude: 0},
				{Latitude: 0.0006, Longitude: 0.00003},
				{Latitude: 0.0009, Longitude: 0.00012},
				{Latitude: 0.0012, Longitude: 0.00027},
				{Latitude: 0.0015, Longitude: 0.00048},
			},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "CURVE" {
		t.Fatalf("expected CURVE event, got %q", events[0].Type)
	}
	if events[0].Metadata["direction"] != "right" {
		t.Fatalf("expected right curve, got %v", events[0].Metadata["direction"])
	}
	if events[0].RouteDistanceMeters <= 0 {
		t.Fatalf("expected route distance to be set, got %d", events[0].RouteDistanceMeters)
	}
}

func TestAnalyzeCurvesDetectsSharpCurve(t *testing.T) {
	events := AnalyzeCurves([]RouteSegmentResult{
		{
			Sequence:       2,
			DistanceMeters: 140,
			Geometry: []Coordinate{
				{Latitude: 0, Longitude: 0},
				{Latitude: 0.0003, Longitude: 0},
				{Latitude: 0.0005, Longitude: 0.00012},
				{Latitude: 0.00055, Longitude: 0.00035},
				{Latitude: 0.00045, Longitude: 0.00058},
				{Latitude: 0.0002, Longitude: 0.0007},
			},
		},
	})

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != "SHARP_CURVE" {
		t.Fatalf("expected SHARP_CURVE event, got %q", events[0].Type)
	}
	if events[0].SegmentSequence != 2 {
		t.Fatalf("expected segment sequence 2, got %d", events[0].SegmentSequence)
	}
}

func TestAnalyzeCurvesDetectsCurveSequence(t *testing.T) {
	events := AnalyzeCurves([]RouteSegmentResult{
		curvedSegment(0, 0, 100),
		curvedSegment(1, 0.0012, 100),
		curvedSegment(2, 0.0024, 100),
	})

	var curveCount int
	var sequenceCount int
	for _, event := range events {
		switch event.Type {
		case "CURVE":
			curveCount++
		case "CURVE_SEQUENCE":
			sequenceCount++
		}
	}

	if curveCount != 3 {
		t.Fatalf("expected 3 curve events, got %d", curveCount)
	}
	if sequenceCount != 1 {
		t.Fatalf("expected 1 curve sequence event, got %d", sequenceCount)
	}
}

func TestAnalyzeCurvesIgnoresSegmentBoundaryTurn(t *testing.T) {
	events := AnalyzeCurves([]RouteSegmentResult{
		{
			Sequence:       0,
			DistanceMeters: 100,
			Geometry: []Coordinate{
				{Latitude: 0, Longitude: 0},
				{Latitude: 0.0004, Longitude: 0},
				{Latitude: 0.0004, Longitude: 0.0004},
				{Latitude: 0.0004, Longitude: 0.0008},
			},
		},
	})

	if len(events) != 0 {
		t.Fatalf("expected no curve events at segment boundary, got %d", len(events))
	}
}

func curvedSegment(sequence int, latitudeOffset float64, distanceMeters int) RouteSegmentResult {
	return RouteSegmentResult{
		Sequence:       sequence,
		DistanceMeters: distanceMeters,
		Geometry: []Coordinate{
			{Latitude: latitudeOffset, Longitude: 0},
			{Latitude: latitudeOffset + 0.00025, Longitude: 0},
			{Latitude: latitudeOffset + 0.0005, Longitude: 0.00002},
			{Latitude: latitudeOffset + 0.00075, Longitude: 0.00008},
			{Latitude: latitudeOffset + 0.001, Longitude: 0.00018},
			{Latitude: latitudeOffset + 0.00125, Longitude: 0.00032},
		},
	}
}
