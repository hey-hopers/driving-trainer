package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"driving-trainer/backend/internal/routes"
)

func testRouter() http.Handler {
	return NewRouter(routes.NewHandler(routes.NewService()))
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
	if response.Route.DistanceMeters != 5421 {
		t.Fatalf("expected distance 5421, got %d", response.Route.DistanceMeters)
	}
	if response.Analysis.Categories.Hills != 6.2 {
		t.Fatalf("expected hills score 6.2, got %v", response.Analysis.Categories.Hills)
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
