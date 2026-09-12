package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"driving-trainer/backend/internal/routes"
)

func testRouter() http.Handler {
	return NewRouter(routes.NewHandler(routes.NewService(fakeRouter{}, fakeStore{})))
}

type fakeRouter struct{}

func (fakeRouter) CalculateRoute(ctx context.Context, origin routes.Coordinate, destination routes.Coordinate) (routes.RouteResult, error) {
	return routes.RouteResult{
		DistanceMeters:  2476,
		DurationSeconds: 376,
		Polyline:        "??AA",
	}, nil
}

type fakeStore struct{}

func (fakeStore) CreateRoute(ctx context.Context, route routes.NewRoute) (routes.Route, error) {
	return routes.Route{
		ID:              "11111111-1111-1111-1111-111111111111",
		Source:          route.Source,
		Geometry:        route.Geometry,
		DistanceMeters:  route.DistanceMeters,
		DurationSeconds: route.DurationSeconds,
		Polyline:        route.Polyline,
		CreatedAt:       time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC),
	}, nil
}

func (fakeStore) GetRoute(ctx context.Context, id string) (routes.Route, error) {
	if id == "22222222-2222-2222-2222-222222222222" {
		return routes.Route{}, routes.ErrRouteNotFound
	}
	if id == "33333333-3333-3333-3333-333333333333" {
		return routes.Route{}, errors.New("database unavailable")
	}
	return routes.Route{
		ID:              id,
		Source:          "valhalla",
		Geometry:        []routes.Coordinate{{Latitude: 0, Longitude: 0}, {Latitude: 0.000001, Longitude: 0.000001}},
		DistanceMeters:  2476,
		DurationSeconds: 376,
		Polyline:        "??AA",
		CreatedAt:       time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC),
	}, nil
}

func TestAnalyzeRouteValidRequest(t *testing.T) {
	body := []byte(`{
		"origin": {"latitude": -26.9194, "longitude": -49.0661},
		"destination": {"latitude": -26.9050, "longitude": -49.0750}
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/routes/analyze", bytes.NewReader(body))
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var response routes.AnalyzeRouteResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}
	if response.Route.DistanceMeters != 2476 {
		t.Fatalf("expected distance 2476, got %d", response.Route.DistanceMeters)
	}
	if response.Route.DurationSeconds != 376 {
		t.Fatalf("expected duration 376, got %d", response.Route.DurationSeconds)
	}
	if response.Route.ID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected persisted route id, got %q", response.Route.ID)
	}
	if response.Route.Polyline != "??AA" {
		t.Fatalf("expected polyline ??AA, got %q", response.Route.Polyline)
	}
}

func TestGetRouteValidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/11111111-1111-1111-1111-111111111111", nil)
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var response routes.GetRouteResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}
	if response.Route.ID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("expected route id, got %q", response.Route.ID)
	}
	if response.Route.Source != "valhalla" {
		t.Fatalf("expected source valhalla, got %q", response.Route.Source)
	}
	if len(response.Route.Geometry) != 2 {
		t.Fatalf("expected 2 geometry points, got %d", len(response.Route.Geometry))
	}
}

func TestGetRouteInvalidID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/not-a-uuid", nil)
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}

func TestGetRouteNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/22222222-2222-2222-2222-222222222222", nil)
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, res.Code)
	}
}

func TestGetRouteDatabaseUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/routes/33333333-3333-3333-3333-333333333333", nil)
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, res.Code)
	}
}

func TestAnalyzeRouteInvalidLatitude(t *testing.T) {
	body := []byte(`{
		"origin": {"latitude": -91, "longitude": -49.0661},
		"destination": {"latitude": -26.9050, "longitude": -49.0750}
	}`)

	assertBadRequest(t, body)
}

func TestAnalyzeRouteInvalidLongitude(t *testing.T) {
	body := []byte(`{
		"origin": {"latitude": -26.9194, "longitude": -181},
		"destination": {"latitude": -26.9050, "longitude": -49.0750}
	}`)

	assertBadRequest(t, body)
}

func TestAnalyzeRouteSameOriginAndDestination(t *testing.T) {
	body := []byte(`{
		"origin": {"latitude": -26.9194, "longitude": -49.0661},
		"destination": {"latitude": -26.9194, "longitude": -49.0661}
	}`)

	assertBadRequest(t, body)
}

func TestAnalyzeRouteInvalidJSON(t *testing.T) {
	body := []byte(`{"origin":`)

	assertBadRequest(t, body)
}

func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		t.Fatalf("expected valid JSON response: %v", err)
	}
	if response["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", response["status"])
	}
}

func assertBadRequest(t *testing.T, body []byte) {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/routes/analyze", bytes.NewReader(body))
	res := httptest.NewRecorder()

	testRouter().ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, res.Code)
	}
}
