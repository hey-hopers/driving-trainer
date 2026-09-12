package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"driving-trainer/backend/internal/config"
	db "driving-trainer/backend/internal/db/sqlc"
	"driving-trainer/backend/internal/httpserver"
	"driving-trainer/backend/internal/routes"
	"driving-trainer/backend/internal/routing/valhalla"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("create database pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	routerClient := valhalla.NewClient(cfg.ValhallaURL, &http.Client{
		Timeout: 10 * time.Second,
	})
	routeStore := routes.NewPostgresStore(db.New(pool))
	routeService := routes.NewService(routerClient, routeStore)
	routeHandler := routes.NewHandler(routeService)
	router := httpserver.NewRouter(routeHandler)

	log.Printf("API listening on http://localhost%s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatal(err)
	}
}
