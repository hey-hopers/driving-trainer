package routes

import (
	"errors"
	"math"
)

const polyline6Precision = 1_000_000

func DecodePolyline6(encoded string) ([]Coordinate, error) {
	if encoded == "" {
		return nil, errors.New("polyline is required")
	}

	var coordinates []Coordinate
	var latitude int
	var longitude int

	for index := 0; index < len(encoded); {
		latDelta, nextIndex, err := decodePolylineValue(encoded, index)
		if err != nil {
			return nil, err
		}
		index = nextIndex

		lonDelta, nextIndex, err := decodePolylineValue(encoded, index)
		if err != nil {
			return nil, err
		}
		index = nextIndex

		latitude += latDelta
		longitude += lonDelta

		coordinates = append(coordinates, Coordinate{
			Latitude:  float64(latitude) / polyline6Precision,
			Longitude: float64(longitude) / polyline6Precision,
		})
	}

	if len(coordinates) < 2 {
		return nil, errors.New("route geometry must contain at least two points")
	}

	return coordinates, nil
}

func decodePolylineValue(encoded string, index int) (int, int, error) {
	var result int
	var shift uint

	for {
		if index >= len(encoded) {
			return 0, index, errors.New("invalid polyline encoding")
		}

		value := int(encoded[index]) - 63
		if value < 0 {
			return 0, index, errors.New("invalid polyline character")
		}

		index++
		result |= (value & 0x1f) << shift
		shift += 5

		if value < 0x20 {
			break
		}
	}

	if result&1 == 1 {
		return ^(result >> 1), index, nil
	}

	return result >> 1, index, nil
}

func IsValidUUID(value string) bool {
	if len(value) != 36 {
		return false
	}

	for i, r := range value {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if !isHex(r) {
				return false
			}
		}
	}

	return true
}

func coordinatesEqual(a, b Coordinate) bool {
	return math.Abs(a.Latitude-b.Latitude) < 0.000001 &&
		math.Abs(a.Longitude-b.Longitude) < 0.000001
}

func isHex(r rune) bool {
	return (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'f') ||
		(r >= 'A' && r <= 'F')
}
