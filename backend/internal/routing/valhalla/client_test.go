package valhalla

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"driving-trainer/backend/internal/routes"
)

func TestCalculateRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected method POST, got %s", r.Method)
		}

		switch r.URL.Path {
		case "/route":
			handleRouteTestRequest(t, w, r)
		case "/trace_attributes":
			handleTraceAttributesTestRequest(t, w, r)
		case "/height":
			handleHeightTestRequest(t, w, r)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())

	result, err := client.CalculateRoute(
		context.Background(),
		routes.Coordinate{Latitude: -26.9194, Longitude: -49.0661},
		routes.Coordinate{Latitude: -26.9050, Longitude: -49.0750},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.DistanceMeters != 2476 {
		t.Fatalf("expected distance 2476, got %d", result.DistanceMeters)
	}
	if result.DurationSeconds != 376 {
		t.Fatalf("expected duration 376, got %d", result.DurationSeconds)
	}
	if result.Polyline != "??AA" {
		t.Fatalf("expected polyline ??AA, got %q", result.Polyline)
	}
	if len(result.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(result.Segments))
	}
	if result.Segments[0].Sequence != 0 {
		t.Fatalf("expected segment sequence 0, got %d", result.Segments[0].Sequence)
	}
	if result.Segments[0].RoadName != "Rua Teste" {
		t.Fatalf("expected road name Rua Teste, got %q", result.Segments[0].RoadName)
	}
	if result.Segments[0].RoadClass != "residential" {
		t.Fatalf("expected road class residential, got %q", result.Segments[0].RoadClass)
	}
	if result.Segments[0].RoadUse != "road" {
		t.Fatalf("expected road use road, got %q", result.Segments[0].RoadUse)
	}
	if result.Segments[0].SpeedLimitKph != 40 {
		t.Fatalf("expected speed limit 40, got %d", result.Segments[0].SpeedLimitKph)
	}
	if len(result.ElevationProfile) != 2 {
		t.Fatalf("expected 2 elevation samples, got %d", len(result.ElevationProfile))
	}
	if result.ElevationProfile[1].ElevationMeters != 18 {
		t.Fatalf("expected second elevation 18, got %f", result.ElevationProfile[1].ElevationMeters)
	}
}

func TestCalculateRouteFallsBackToManeuversWhenTraceAttributesFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/route":
			handleRouteTestRequest(t, w, r)
		case "/trace_attributes":
			w.WriteHeader(http.StatusBadRequest)
		case "/height":
			handleHeightTestRequest(t, w, r)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())

	result, err := client.CalculateRoute(
		context.Background(),
		routes.Coordinate{Latitude: -26.9194, Longitude: -49.0661},
		routes.Coordinate{Latitude: -26.9050, Longitude: -49.0750},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result.Segments) != 1 {
		t.Fatalf("expected 1 fallback segment, got %d", len(result.Segments))
	}
	if result.Segments[0].RoadName != "Rua Teste" {
		t.Fatalf("expected fallback road name Rua Teste, got %q", result.Segments[0].RoadName)
	}
	if result.Segments[0].RoadClass != "" {
		t.Fatalf("expected empty fallback road class, got %q", result.Segments[0].RoadClass)
	}
}

func handleRouteTestRequest(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()

	var request routeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.Fatalf("expected valid request JSON: %v", err)
	}
	if request.Costing != "auto" {
		t.Fatalf("expected costing auto, got %q", request.Costing)
	}
	if len(request.Locations) != 2 {
		t.Fatalf("expected 2 locations, got %d", len(request.Locations))
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{
		"trip": {
			"summary": {
				"length": 2.476,
				"time": 376.187
			},
			"legs": [
				{
					"shape": "??AA",
					"maneuvers": [
						{
							"begin_shape_index": 0,
							"end_shape_index": 1,
							"length": 2.476,
							"time": 376.187,
							"street_names": ["Rua Teste"]
						}
					]
				}
			]
		}
	}`))
}

func handleTraceAttributesTestRequest(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()

	var request traceAttributesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.Fatalf("expected valid trace attributes request JSON: %v", err)
	}
	if request.EncodedPolyline != "??AA" {
		t.Fatalf("expected encoded polyline ??AA, got %q", request.EncodedPolyline)
	}
	if request.ShapeMatch != "walk_or_snap" {
		t.Fatalf("expected shape match walk_or_snap, got %q", request.ShapeMatch)
	}
	if request.Costing != "auto" {
		t.Fatalf("expected costing auto, got %q", request.Costing)
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{
		"edges": [
			{
				"names": ["Rua Teste"],
				"length": 2.476,
				"speed": 24,
				"road_class": "residential",
				"use": "road",
				"speed_limit": 40,
				"begin_shape_index": 0,
				"end_shape_index": 1
			}
		]
	}`))
}

func handleHeightTestRequest(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()

	var request heightRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.Fatalf("expected valid height request JSON: %v", err)
	}
	if request.EncodedPolyline != "??AA" {
		t.Fatalf("expected encoded polyline ??AA, got %q", request.EncodedPolyline)
	}
	if request.ShapeFormat != "polyline6" {
		t.Fatalf("expected shape format polyline6, got %q", request.ShapeFormat)
	}
	if !request.Range {
		t.Fatal("expected range height request")
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{
		"range_height": [
			[0, 10],
			[2476, 18]
		]
	}`))
}

func TestCalculateRouteValhallaError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"No path could be found"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, server.Client())

	_, err := client.CalculateRoute(
		context.Background(),
		routes.Coordinate{Latitude: -26.9194, Longitude: -49.0661},
		routes.Coordinate{Latitude: -26.9050, Longitude: -49.0750},
	)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
