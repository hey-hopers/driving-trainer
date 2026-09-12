package routes

import (
	"testing"
)

func TestLineStringWKT(t *testing.T) {
	wkt, err := lineStringWKT([]Coordinate{
		{Latitude: -26.9194, Longitude: -49.0661},
		{Latitude: -26.905, Longitude: -49.075},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "LINESTRING(-49.0661 -26.9194,-49.075 -26.905)"
	if wkt != expected {
		t.Fatalf("expected WKT %q, got %q", expected, wkt)
	}
}

func TestLineStringWKTRequiresTwoPoints(t *testing.T) {
	_, err := lineStringWKT([]Coordinate{{Latitude: 0, Longitude: 0}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCoordinatesFromGeoJSON(t *testing.T) {
	coordinates, err := coordinatesFromGeoJSON(`{
		"type": "LineString",
		"coordinates": [[-49.0661, -26.9194], [-49.075, -26.905]]
	}`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []Coordinate{
		{Latitude: -26.9194, Longitude: -49.0661},
		{Latitude: -26.905, Longitude: -49.075},
	}
	for i := range expected {
		if !coordinatesEqual(coordinates[i], expected[i]) {
			t.Fatalf("expected coordinate %d to be %+v, got %+v", i, expected[i], coordinates[i])
		}
	}
}
