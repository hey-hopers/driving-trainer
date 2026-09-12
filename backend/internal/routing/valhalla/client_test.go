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
		if r.URL.Path != "/route" {
			t.Fatalf("expected path /route, got %s", r.URL.Path)
		}

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
						"shape": "encoded-polyline6"
					}
				]
			}
		}`))
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
	if result.Polyline != "encoded-polyline6" {
		t.Fatalf("expected polyline encoded-polyline6, got %q", result.Polyline)
	}
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
