package routes

import (
	"reflect"
	"testing"
)

func TestAnalyzeDifficultyDeterministicBreakdown(t *testing.T) {
	incline := 9.0
	segments := []RouteSegmentResult{
		{
			Sequence:       0,
			DistanceMeters: 500,
			RoadClass:      "residential",
		},
		{
			Sequence:       1,
			DistanceMeters: 500,
			RoadClass:      "motorway",
			RoadUse:        "highway",
			SpeedLimitKph:  80,
			InclineMaxPct:  &incline,
		},
	}
	events := []RouteEventResult{
		{Type: "HILL_STOP", RouteDistanceMeters: 100, DifficultyScore: 8.7},
		{Type: "SHARP_CURVE", RouteDistanceMeters: 300, DifficultyScore: 5.5},
		{Type: "HIGHWAY_ENTRY", RouteDistanceMeters: 500, DifficultyScore: 6},
		{Type: "INTERSECTION", RouteDistanceMeters: 850, DifficultyScore: 3},
	}

	first := AnalyzeDifficulty(segments, events, 1000)
	second := AnalyzeDifficulty(segments, events, 1000)

	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected deterministic analysis, got %+v and %+v", first, second)
	}
	if first.EngineVersion != "difficulty-v1" {
		t.Fatalf("expected engine version difficulty-v1, got %q", first.EngineVersion)
	}
	if first.OverallDifficulty != 6.93 {
		t.Fatalf("expected overall difficulty 6.93, got %v", first.OverallDifficulty)
	}
	if first.AverageDifficulty != 5.69 {
		t.Fatalf("expected average difficulty 5.69, got %v", first.AverageDifficulty)
	}
	if first.PeakDifficulty != 9.2 {
		t.Fatalf("expected peak difficulty 9.2, got %v", first.PeakDifficulty)
	}
	if first.ComplexityScore != 7.66 {
		t.Fatalf("expected complexity score 7.66, got %v", first.ComplexityScore)
	}
	if first.CategoryScores["hills"] != 8.7 {
		t.Fatalf("expected hills score 8.7, got %v", first.CategoryScores["hills"])
	}
	if first.Categories.HighSpeed != 6 {
		t.Fatalf("expected legacy high speed score 6, got %v", first.Categories.HighSpeed)
	}
}

func TestAnalyzeDifficultyEmptyRoute(t *testing.T) {
	analysis := AnalyzeDifficulty(nil, nil, 0)

	if analysis.OverallDifficulty != 0 {
		t.Fatalf("expected zero overall difficulty, got %v", analysis.OverallDifficulty)
	}
	if analysis.PeakDifficulty != 0 {
		t.Fatalf("expected zero peak difficulty, got %v", analysis.PeakDifficulty)
	}
	if analysis.EngineVersion != "difficulty-v1" {
		t.Fatalf("expected engine version difficulty-v1, got %q", analysis.EngineVersion)
	}
}
