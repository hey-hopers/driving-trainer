package valhalla

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"driving-trainer/backend/internal/routes"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *Client) CalculateRoute(ctx context.Context, origin routes.Coordinate, destination routes.Coordinate) (routes.RouteResult, error) {
	body, err := json.Marshal(routeRequest{
		Locations: []location{
			{Lat: origin.Latitude, Lon: origin.Longitude, Type: "break"},
			{Lat: destination.Latitude, Lon: destination.Longitude, Type: "break"},
		},
		Costing: "auto",
		DirectionsOptions: directionsOptions{
			Units: "kilometers",
		},
	})
	if err != nil {
		return routes.RouteResult{}, fmt.Errorf("marshal valhalla route request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/route", bytes.NewReader(body))
	if err != nil {
		return routes.RouteResult{}, fmt.Errorf("create valhalla route request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return routes.RouteResult{}, fmt.Errorf("call valhalla route: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		var errorResponse routeErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&errorResponse); err == nil && errorResponse.Error != "" {
			return routes.RouteResult{}, fmt.Errorf("valhalla route failed: status %d: %s", res.StatusCode, errorResponse.Error)
		}
		return routes.RouteResult{}, fmt.Errorf("valhalla route failed: status %d", res.StatusCode)
	}

	var route routeResponse
	if err := json.NewDecoder(res.Body).Decode(&route); err != nil {
		return routes.RouteResult{}, fmt.Errorf("decode valhalla route response: %w", err)
	}

	if len(route.Trip.Legs) == 0 || route.Trip.Legs[0].Shape == "" {
		return routes.RouteResult{}, errors.New("valhalla route response missing shape")
	}

	leg := route.Trip.Legs[0]
	geometry, err := routes.DecodePolyline6(leg.Shape)
	if err != nil {
		return routes.RouteResult{}, fmt.Errorf("decode valhalla route shape: %w", err)
	}
	segments := []routes.RouteSegmentResult{}
	edges, err := c.traceAttributes(ctx, leg.Shape)
	if err == nil {
		segments, err = segmentsFromEdges(geometry, edges)
		if err != nil {
			return routes.RouteResult{}, err
		}
	}
	if len(segments) == 0 {
		segments, err = segmentsFromManeuvers(geometry, leg.Maneuvers)
		if err != nil {
			return routes.RouteResult{}, err
		}
	}

	elevationProfile, err := c.height(ctx, leg.Shape)
	if err != nil {
		elevationProfile = nil
	}

	return routes.RouteResult{
		DistanceMeters:   int(math.Round(route.Trip.Summary.Length * 1000)),
		DurationSeconds:  int(math.Round(route.Trip.Summary.Time)),
		Polyline:         leg.Shape,
		Segments:         segments,
		ElevationProfile: elevationProfile,
	}, nil
}

type routeRequest struct {
	Locations         []location        `json:"locations"`
	Costing           string            `json:"costing"`
	DirectionsOptions directionsOptions `json:"directions_options"`
}

type location struct {
	Lat  float64 `json:"lat"`
	Lon  float64 `json:"lon"`
	Type string  `json:"type"`
}

type directionsOptions struct {
	Units string `json:"units"`
}

type routeResponse struct {
	Trip trip `json:"trip"`
}

type trip struct {
	Summary summary `json:"summary"`
	Legs    []leg   `json:"legs"`
}

type summary struct {
	Length float64 `json:"length"`
	Time   float64 `json:"time"`
}

type leg struct {
	Shape     string     `json:"shape"`
	Maneuvers []maneuver `json:"maneuvers"`
}

type maneuver struct {
	BeginShapeIndex int      `json:"begin_shape_index"`
	EndShapeIndex   int      `json:"end_shape_index"`
	Length          float64  `json:"length"`
	Time            float64  `json:"time"`
	StreetNames     []string `json:"street_names"`
}

type routeErrorResponse struct {
	Error string `json:"error"`
}

type heightRequest struct {
	EncodedPolyline  string  `json:"encoded_polyline"`
	ShapeFormat      string  `json:"shape_format"`
	Range            bool    `json:"range"`
	ResampleDistance float64 `json:"resample_distance"`
	HeightPrecision  int     `json:"height_precision"`
}

type heightResponse struct {
	RangeHeight [][]*float64 `json:"range_height"`
}

type traceAttributesRequest struct {
	EncodedPolyline string           `json:"encoded_polyline"`
	ShapeMatch      string           `json:"shape_match"`
	Costing         string           `json:"costing"`
	Filters         attributeFilters `json:"filters"`
}

type attributeFilters struct {
	Action     string   `json:"action"`
	Attributes []string `json:"attributes"`
}

type traceAttributesResponse struct {
	Edges []edgeAttribute `json:"edges"`
}

type edgeAttribute struct {
	Names           []string `json:"names"`
	Length          float64  `json:"length"`
	Speed           float64  `json:"speed"`
	RoadClass       string   `json:"road_class"`
	RoadUse         string   `json:"use"`
	SpeedLimitKph   int      `json:"speed_limit"`
	BeginShapeIndex int      `json:"begin_shape_index"`
	EndShapeIndex   int      `json:"end_shape_index"`
}

func (c *Client) traceAttributes(ctx context.Context, encodedPolyline string) ([]edgeAttribute, error) {
	body, err := json.Marshal(traceAttributesRequest{
		EncodedPolyline: encodedPolyline,
		ShapeMatch:      "walk_or_snap",
		Costing:         "auto",
		Filters: attributeFilters{
			Action: "include",
			Attributes: []string{
				"edge.names",
				"edge.length",
				"edge.speed",
				"edge.road_class",
				"edge.use",
				"edge.speed_limit",
				"edge.begin_shape_index",
				"edge.end_shape_index",
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal valhalla trace attributes request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/trace_attributes", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create valhalla trace attributes request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call valhalla trace attributes: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		var errorResponse routeErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&errorResponse); err == nil && errorResponse.Error != "" {
			return nil, fmt.Errorf("valhalla trace attributes failed: status %d: %s", res.StatusCode, errorResponse.Error)
		}
		return nil, fmt.Errorf("valhalla trace attributes failed: status %d", res.StatusCode)
	}

	var attributes traceAttributesResponse
	if err := json.NewDecoder(res.Body).Decode(&attributes); err != nil {
		return nil, fmt.Errorf("decode valhalla trace attributes response: %w", err)
	}

	return attributes.Edges, nil
}

func (c *Client) height(ctx context.Context, encodedPolyline string) ([]routes.ElevationSample, error) {
	body, err := json.Marshal(heightRequest{
		EncodedPolyline:  encodedPolyline,
		ShapeFormat:      "polyline6",
		Range:            true,
		ResampleDistance: 25,
		HeightPrecision:  1,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal valhalla height request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/height", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create valhalla height request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call valhalla height: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		var errorResponse routeErrorResponse
		if err := json.NewDecoder(res.Body).Decode(&errorResponse); err == nil && errorResponse.Error != "" {
			return nil, fmt.Errorf("valhalla height failed: status %d: %s", res.StatusCode, errorResponse.Error)
		}
		return nil, fmt.Errorf("valhalla height failed: status %d", res.StatusCode)
	}

	var height heightResponse
	if err := json.NewDecoder(res.Body).Decode(&height); err != nil {
		return nil, fmt.Errorf("decode valhalla height response: %w", err)
	}

	samples := make([]routes.ElevationSample, 0, len(height.RangeHeight))
	for i, pair := range height.RangeHeight {
		if len(pair) != 2 || pair[0] == nil || pair[1] == nil {
			return nil, fmt.Errorf("valhalla height sample %d is invalid", i)
		}
		samples = append(samples, routes.ElevationSample{
			RouteDistanceMeters: *pair[0],
			ElevationMeters:     *pair[1],
		})
	}

	return samples, nil
}

func segmentsFromEdges(geometry []routes.Coordinate, edges []edgeAttribute) ([]routes.RouteSegmentResult, error) {
	if len(edges) == 0 {
		return nil, nil
	}

	groups := make([]edgeGroup, 0, len(edges))
	for i, edge := range edges {
		if edge.BeginShapeIndex < 0 ||
			edge.EndShapeIndex < edge.BeginShapeIndex ||
			edge.EndShapeIndex >= len(geometry) {
			return nil, fmt.Errorf("valhalla edge %d has invalid shape indexes", i)
		}
		if edge.EndShapeIndex == edge.BeginShapeIndex {
			continue
		}

		roadName := strings.Join(edge.Names, " / ")
		if len(groups) == 0 || !groups[len(groups)-1].matches(edge, roadName) {
			groups = append(groups, edgeGroup{
				beginShapeIndex: edge.BeginShapeIndex,
				endShapeIndex:   edge.EndShapeIndex,
				roadClass:       edge.RoadClass,
				roadName:        roadName,
				roadUse:         edge.RoadUse,
				speedLimitKph:   edge.SpeedLimitKph,
				distanceMeters:  int(math.Round(edge.Length * 1000)),
				durationSeconds: speedDurationSeconds(edge.Length, edge.Speed),
			})
			continue
		}

		group := &groups[len(groups)-1]
		group.endShapeIndex = edge.EndShapeIndex
		group.distanceMeters += int(math.Round(edge.Length * 1000))
		group.durationSeconds += speedDurationSeconds(edge.Length, edge.Speed)
	}

	segments := make([]routes.RouteSegmentResult, 0, len(groups))
	for _, group := range groups {
		segmentGeometry := append([]routes.Coordinate(nil), geometry[group.beginShapeIndex:group.endShapeIndex+1]...)
		if len(segmentGeometry) < 2 {
			continue
		}
		segments = append(segments, routes.RouteSegmentResult{
			Sequence:        len(segments),
			Geometry:        segmentGeometry,
			DistanceMeters:  group.distanceMeters,
			DurationSeconds: group.durationSeconds,
			RoadClass:       group.roadClass,
			RoadName:        group.roadName,
			RoadUse:         group.roadUse,
			SpeedLimitKph:   group.speedLimitKph,
		})
	}

	return segments, nil
}

type edgeGroup struct {
	beginShapeIndex int
	endShapeIndex   int
	roadClass       string
	roadName        string
	roadUse         string
	speedLimitKph   int
	distanceMeters  int
	durationSeconds int
}

func (g edgeGroup) matches(edge edgeAttribute, roadName string) bool {
	return g.roadClass == edge.RoadClass &&
		g.roadName == roadName &&
		g.roadUse == edge.RoadUse &&
		g.speedLimitKph == edge.SpeedLimitKph &&
		g.endShapeIndex == edge.BeginShapeIndex
}

func speedDurationSeconds(lengthKilometers float64, speedKPH float64) int {
	if lengthKilometers <= 0 || speedKPH <= 0 {
		return 0
	}
	return int(math.Round((lengthKilometers / speedKPH) * 3600))
}

func segmentsFromManeuvers(geometry []routes.Coordinate, maneuvers []maneuver) ([]routes.RouteSegmentResult, error) {
	if len(maneuvers) == 0 {
		return nil, nil
	}

	segments := make([]routes.RouteSegmentResult, 0, len(maneuvers))
	for i, maneuver := range maneuvers {
		if maneuver.BeginShapeIndex < 0 ||
			maneuver.EndShapeIndex < maneuver.BeginShapeIndex ||
			maneuver.EndShapeIndex >= len(geometry) {
			return nil, fmt.Errorf("valhalla maneuver %d has invalid shape indexes", i)
		}

		segmentGeometry := append([]routes.Coordinate(nil), geometry[maneuver.BeginShapeIndex:maneuver.EndShapeIndex+1]...)
		if len(segmentGeometry) < 2 {
			continue
		}

		segments = append(segments, routes.RouteSegmentResult{
			Sequence:        len(segments),
			Geometry:        segmentGeometry,
			DistanceMeters:  int(math.Round(maneuver.Length * 1000)),
			DurationSeconds: int(math.Round(maneuver.Time)),
			RoadName:        strings.Join(maneuver.StreetNames, " / "),
		})
	}

	return segments, nil
}
