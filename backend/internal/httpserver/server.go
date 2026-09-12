package httpserver

import (
	"encoding/json"
	"net/http"

	"driving-trainer/backend/internal/routes"

	"github.com/go-chi/chi/v5"
)

func NewRouter(routeHandler *routes.Handler) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", healthHandler)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/routes/analyze", routeHandler.Analyze)
	})

	return r
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
