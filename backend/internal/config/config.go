package config

import "os"

type Config struct {
	Addr        string
	ValhallaURL string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	valhallaURL := os.Getenv("VALHALLA_URL")
	if valhallaURL == "" {
		valhallaURL = "http://localhost:8002"
	}

	return Config{
		Addr:        ":" + port,
		ValhallaURL: valhallaURL,
	}
}
