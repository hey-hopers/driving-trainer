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

	return routes.RouteResult{
		DistanceMeters:  int(math.Round(route.Trip.Summary.Length * 1000)),
		DurationSeconds: int(math.Round(route.Trip.Summary.Time)),
		Polyline:        route.Trip.Legs[0].Shape,
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
	Shape string `json:"shape"`
}

type routeErrorResponse struct {
	Error string `json:"error"`
}
