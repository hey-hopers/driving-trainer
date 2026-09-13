package routes

import (
	"math"
	"strings"
)

const difficultyEngineVersionV1 = "difficulty-v1"

func AnalyzeDifficulty(segments []RouteSegmentResult, events []RouteEventResult, routeDistanceMeters int) RouteAnalysis {
	categoryScores := map[string]float64{
		"hills":          hillCategoryScore(segments, events),
		"curves":         eventCategoryScore(events, []string{"CURVE", "SHARP_CURVE", "CURVE_SEQUENCE"}),
		"intersections":  eventCategoryScore(events, []string{"INTERSECTION", "COMPLEX_INTERSECTION", "ROUNDABOUT", "STOP", "TRAFFIC_LIGHT"}),
		"highSpeed":      highSpeedCategoryScore(segments, events),
		"compound":       eventCategoryScore(events, []string{"HILL_STOP"}),
		"eventDensity":   eventDensityScore(events, routeDistanceMeters),
		"eventDiversity": eventDiversityScore(events),
	}

	averageDifficulty := averageDifficultyScore(segments, events, routeDistanceMeters)
	peakDifficulty := peakDifficultyScore(segments, events)
	complexityScore := complexityScore(categoryScores, events)
	overallDifficulty := round(clamp(
		averageDifficulty*0.35+
			peakDifficulty*0.25+
			complexityScore*0.25+
			categoryScores["eventDiversity"]*0.15,
		0, 10), 2)

	return RouteAnalysis{
		EngineVersion:     difficultyEngineVersionV1,
		Difficulty:        overallDifficulty,
		OverallDifficulty: overallDifficulty,
		AverageDifficulty: averageDifficulty,
		PeakDifficulty:    peakDifficulty,
		ComplexityScore:   complexityScore,
		CategoryScores:    categoryScores,
		Categories: RouteCategoryScores{
			Hills:         categoryScores["hills"],
			Curves:        categoryScores["curves"],
			Intersections: categoryScores["intersections"],
			HighSpeed:     categoryScores["highSpeed"],
		},
	}
}

func averageDifficultyScore(segments []RouteSegmentResult, events []RouteEventResult, routeDistanceMeters int) float64 {
	segmentWeightedSum := 0.0
	segmentDistance := 0
	for _, segment := range segments {
		distance := segment.DistanceMeters
		if distance <= 0 {
			distance = 1
		}
		segmentWeightedSum += segmentDifficultyScore(segment) * float64(distance)
		segmentDistance += distance
	}

	segmentAverage := 0.0
	if segmentDistance > 0 {
		segmentAverage = segmentWeightedSum / float64(segmentDistance)
	}

	eventAverage := 0.0
	if len(events) > 0 {
		for _, event := range events {
			eventAverage += eventDifficultyScore(event)
		}
		eventAverage = eventAverage / float64(len(events))
	}

	density := eventDensityScore(events, routeDistanceMeters)
	switch {
	case segmentAverage == 0 && eventAverage == 0:
		return 0
	case segmentAverage == 0:
		return round(clamp(eventAverage*0.85+density*0.15, 0, 10), 2)
	case eventAverage == 0:
		return round(clamp(segmentAverage, 0, 10), 2)
	default:
		return round(clamp(segmentAverage*0.55+eventAverage*0.35+density*0.10, 0, 10), 2)
	}
}

func peakDifficultyScore(segments []RouteSegmentResult, events []RouteEventResult) float64 {
	peak := 0.0
	for _, segment := range segments {
		peak = math.Max(peak, segmentDifficultyScore(segment))
	}
	for _, event := range events {
		peak = math.Max(peak, eventDifficultyScore(event))
	}
	return round(clamp(peak, 0, 10), 2)
}

func complexityScore(categoryScores map[string]float64, events []RouteEventResult) float64 {
	compound := categoryScores["compound"]
	curves := categoryScores["curves"]
	intersections := categoryScores["intersections"]
	hills := categoryScores["hills"]
	highSpeed := categoryScores["highSpeed"]
	density := categoryScores["eventDensity"]
	diversity := categoryScores["eventDiversity"]

	compoundMixBonus := 0.0
	if hills > 0 && intersections > 0 {
		compoundMixBonus += 0.7
	}
	if curves > 0 && highSpeed > 0 {
		compoundMixBonus += 0.5
	}
	if hasEventType(events, "COMPLEX_INTERSECTION") && (hills > 0 || curves > 0) {
		compoundMixBonus += 0.6
	}

	return round(clamp(
		compound*0.25+
			intersections*0.20+
			curves*0.15+
			hills*0.15+
			highSpeed*0.10+
			density*0.10+
			diversity*0.05+
			compoundMixBonus,
		0, 10), 2)
}

func segmentDifficultyScore(segment RouteSegmentResult) float64 {
	score := 1.0
	score += inclineContribution(segment)
	score += speedContribution(segment.SpeedLimitKph)

	roadUse := strings.ToLower(segment.RoadUse)
	roadClass := strings.ToLower(segment.RoadClass)
	if strings.Contains(roadUse, "ramp") {
		score += 1.5
	}
	if isHighway(roadUse, roadClass) {
		score += 2.0
	}
	if strings.Contains(roadClass, "residential") || strings.Contains(roadClass, "service") {
		score += 0.5
	}

	return round(clamp(score, 0, 10), 2)
}

func eventDifficultyScore(event RouteEventResult) float64 {
	if event.DifficultyScore > 0 {
		return clamp(event.DifficultyScore, 0, 10)
	}

	switch event.Type {
	case "HILL_STOP":
		return 8.5
	case "STEEP_HILL":
		return 7.5
	case "COMPLEX_INTERSECTION":
		return 6.5
	case "HIGHWAY_ENTRY":
		return 6
	case "SHARP_CURVE", "CURVE_SEQUENCE":
		return 5.5
	case "HIGHWAY_EXIT":
		return 5.5
	case "ROUNDABOUT":
		return 5
	case "TRAFFIC_LIGHT":
		return 4
	case "HILL":
		return 4
	case "STOP":
		return 3.5
	case "INTERSECTION":
		return 3
	case "CURVE":
		return 3
	default:
		return 1
	}
}

func hillCategoryScore(segments []RouteSegmentResult, events []RouteEventResult) float64 {
	score := eventCategoryScore(events, []string{"HILL", "STEEP_HILL", "HILL_STOP"})
	for _, segment := range segments {
		score = math.Max(score, inclineContribution(segment)*1.7)
	}
	return round(clamp(score, 0, 10), 2)
}

func highSpeedCategoryScore(segments []RouteSegmentResult, events []RouteEventResult) float64 {
	score := eventCategoryScore(events, []string{"HIGHWAY_ENTRY", "HIGHWAY_EXIT"})
	for _, segment := range segments {
		score = math.Max(score, speedContribution(segment.SpeedLimitKph)*2.2)
		if isHighway(segment.RoadUse, segment.RoadClass) {
			score = math.Max(score, 6)
		}
	}
	return round(clamp(score, 0, 10), 2)
}

func eventCategoryScore(events []RouteEventResult, eventTypes []string) float64 {
	typeSet := map[string]bool{}
	for _, eventType := range eventTypes {
		typeSet[eventType] = true
	}

	score := 0.0
	count := 0
	for _, event := range events {
		if !typeSet[event.Type] {
			continue
		}
		score = math.Max(score, eventDifficultyScore(event))
		count++
	}
	if count > 1 {
		score += math.Min(float64(count-1)*0.45, 2)
	}
	return round(clamp(score, 0, 10), 2)
}

func eventDensityScore(events []RouteEventResult, routeDistanceMeters int) float64 {
	if len(events) == 0 || routeDistanceMeters <= 0 {
		return 0
	}
	eventsPerKm := float64(len(events)) / (float64(routeDistanceMeters) / 1000)
	return round(clamp(eventsPerKm*1.8, 0, 10), 2)
}

func eventDiversityScore(events []RouteEventResult) float64 {
	types := map[string]bool{}
	for _, event := range events {
		types[event.Type] = true
	}
	return round(clamp(float64(len(types))*1.2, 0, 10), 2)
}

func inclineContribution(segment RouteSegmentResult) float64 {
	incline := 0.0
	if segment.InclineAvgPct != nil {
		incline = math.Max(incline, math.Abs(*segment.InclineAvgPct))
	}
	if segment.InclineMaxPct != nil {
		incline = math.Max(incline, math.Abs(*segment.InclineMaxPct))
	}

	switch {
	case incline >= 12:
		return 5
	case incline >= 8:
		return 4
	case incline >= 5:
		return 2.5
	case incline >= 3:
		return 1.5
	default:
		return 0
	}
}

func speedContribution(speedLimitKph int) float64 {
	switch {
	case speedLimitKph >= 100:
		return 3
	case speedLimitKph >= 80:
		return 2.2
	case speedLimitKph >= 60:
		return 1.2
	default:
		return 0
	}
}

func isHighway(roadUse string, roadClass string) bool {
	roadUse = strings.ToLower(roadUse)
	roadClass = strings.ToLower(roadClass)
	return strings.Contains(roadUse, "motorway") ||
		strings.Contains(roadUse, "highway") ||
		strings.Contains(roadClass, "motorway") ||
		strings.Contains(roadClass, "trunk")
}

func hasEventType(events []RouteEventResult, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func clamp(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
