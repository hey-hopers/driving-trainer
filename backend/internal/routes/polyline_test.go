package routes

import "testing"

func TestDecodePolyline6(t *testing.T) {
	coordinates, err := DecodePolyline6("??AA")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []Coordinate{
		{Latitude: 0, Longitude: 0},
		{Latitude: 0.000001, Longitude: 0.000001},
	}
	if len(coordinates) != len(expected) {
		t.Fatalf("expected %d coordinates, got %d", len(expected), len(coordinates))
	}

	for i := range expected {
		if !coordinatesEqual(coordinates[i], expected[i]) {
			t.Fatalf("expected coordinate %d to be %+v, got %+v", i, expected[i], coordinates[i])
		}
	}
}

func TestDecodePolyline6RequiresAtLeastTwoPoints(t *testing.T) {
	_, err := DecodePolyline6("??")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDecodePolyline6RejectsInvalidEncoding(t *testing.T) {
	_, err := DecodePolyline6("?")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIsValidUUID(t *testing.T) {
	if !IsValidUUID("11111111-1111-1111-1111-111111111111") {
		t.Fatal("expected UUID to be valid")
	}
	if IsValidUUID("not-a-uuid") {
		t.Fatal("expected UUID to be invalid")
	}
}
