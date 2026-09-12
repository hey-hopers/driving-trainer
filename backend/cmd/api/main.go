package main

import (
	"log"
	"net/http"

	"driving-trainer/backend/internal/config"
	"driving-trainer/backend/internal/httpserver"
	"driving-trainer/backend/internal/routes"
)

func main() {
	cfg := config.Load()

	routeService := routes.NewService()
	routeHandler := routes.NewHandler(routeService)
	router := httpserver.NewRouter(routeHandler)

	log.Printf("API listening on http://localhost%s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatal(err)
	}
}
