package main

import (
	"log"
	"net/http"
	"time"

	"driving-trainer/backend/internal/config"
	"driving-trainer/backend/internal/httpserver"
	"driving-trainer/backend/internal/routes"
	"driving-trainer/backend/internal/routing/valhalla"
)

func main() {
	cfg := config.Load()

	routerClient := valhalla.NewClient(cfg.ValhallaURL, &http.Client{
		Timeout: 10 * time.Second,
	})
	routeService := routes.NewService(routerClient)
	routeHandler := routes.NewHandler(routeService)
	router := httpserver.NewRouter(routeHandler)

	log.Printf("API listening on http://localhost%s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatal(err)
	}
}
