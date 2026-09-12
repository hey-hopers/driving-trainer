package routes

import (
	"encoding/json"
	"errors"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeRouteRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	res, err := h.service.Analyze(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to analyze route")
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (r AnalyzeRouteRequest) Validate() error {
	if r.Origin == nil {
		return errors.New("origin is required")
	}
	if r.Destination == nil {
		return errors.New("destination is required")
	}
	if err := validateCoordinate("origin", *r.Origin); err != nil {
		return err
	}
	if err := validateCoordinate("destination", *r.Destination); err != nil {
		return err
	}
	if r.Origin.Latitude == r.Destination.Latitude && r.Origin.Longitude == r.Destination.Longitude {
		return errors.New("origin and destination must be different")
	}

	return nil
}

func validateCoordinate(name string, coordinate Coordinate) error {
	if coordinate.Latitude < -90 || coordinate.Latitude > 90 {
		return errors.New(name + ".latitude must be between -90 and 90")
	}
	if coordinate.Longitude < -180 || coordinate.Longitude > 180 {
		return errors.New(name + ".longitude must be between -180 and 180")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
