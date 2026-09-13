package routes

import "math"

const (
	hillInclineThresholdPercent      = 3.0
	steepHillInclineThresholdPercent = 6.0
)

func AnalyzeElevation(segments []RouteSegmentResult, profile []ElevationSample) ([]RouteSegmentResult, []RouteEventResult) {
	if len(segments) == 0 || len(profile) < 2 {
		return segments, nil
	}

	enriched := append([]RouteSegmentResult(nil), segments...)
	events := []RouteEventResult{}
	segmentStartMeters := 0.0

	for i := range enriched {
		segment := &enriched[i]
		segmentEndMeters := segmentStartMeters + float64(segment.DistanceMeters)
		samples := samplesInRange(profile, segmentStartMeters, segmentEndMeters)
		if len(samples) >= 2 && segment.DistanceMeters > 0 {
			startElevation := samples[0].ElevationMeters
			endElevation := samples[len(samples)-1].ElevationMeters
			avgIncline := ((endElevation - startElevation) / float64(segment.DistanceMeters)) * 100
			maxIncline := maxIntervalIncline(samples)

			segment.ElevationStartM = ptr(round(startElevation, 2))
			segment.ElevationEndM = ptr(round(endElevation, 2))
			segment.InclineAvgPct = ptr(round(avgIncline, 2))
			segment.InclineMaxPct = ptr(round(maxIncline, 2))

			if eventType := hillEventType(maxIncline); eventType != "" {
				events = append(events, RouteEventResult{
					SegmentSequence:     segment.Sequence,
					Type:                eventType,
					Position:            midpoint(segment.Geometry),
					RouteDistanceMeters: int(math.Round((segmentStartMeters + segmentEndMeters) / 2)),
					DifficultyScore:     round(math.Min(math.Abs(maxIncline)*1.2, 10), 2),
					Metadata: map[string]any{
						"direction":             hillDirection(maxIncline),
						"inclineAvgPercent":     round(avgIncline, 2),
						"inclineMaxPercent":     round(maxIncline, 2),
						"elevationStartMeters":  round(startElevation, 2),
						"elevationEndMeters":    round(endElevation, 2),
						"elevationChangeMeters": round(endElevation-startElevation, 2),
					},
				})
			}
		}
		segmentStartMeters = segmentEndMeters
	}

	return enriched, events
}

func samplesInRange(profile []ElevationSample, startMeters float64, endMeters float64) []ElevationSample {
	samples := make([]ElevationSample, 0, len(profile))
	for _, sample := range profile {
		if sample.RouteDistanceMeters >= startMeters && sample.RouteDistanceMeters <= endMeters {
			samples = append(samples, sample)
		}
	}
	return samples
}

func maxIntervalIncline(samples []ElevationSample) float64 {
	maxIncline := 0.0
	for i := 1; i < len(samples); i++ {
		distance := samples[i].RouteDistanceMeters - samples[i-1].RouteDistanceMeters
		if distance <= 0 {
			continue
		}
		incline := ((samples[i].ElevationMeters - samples[i-1].ElevationMeters) / distance) * 100
		if math.Abs(incline) > math.Abs(maxIncline) {
			maxIncline = incline
		}
	}
	return maxIncline
}

func hillEventType(inclinePercent float64) string {
	absoluteIncline := math.Abs(inclinePercent)
	switch {
	case absoluteIncline >= steepHillInclineThresholdPercent:
		return "STEEP_HILL"
	case absoluteIncline >= hillInclineThresholdPercent:
		return "HILL"
	default:
		return ""
	}
}

func hillDirection(inclinePercent float64) string {
	if inclinePercent < 0 {
		return "downhill"
	}
	return "uphill"
}

func midpoint(coordinates []Coordinate) Coordinate {
	if len(coordinates) == 0 {
		return Coordinate{}
	}
	return coordinates[len(coordinates)/2]
}

func ptr(value float64) *float64 {
	return &value
}

func round(value float64, places int) float64 {
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}
