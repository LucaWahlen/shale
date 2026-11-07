package config

import (
	"errors"
	"os"
)

const DefaultDatabaseURL = "postgres://shale:shale@postgres:5432/shale?sslmode=disable"

type Config struct {
	AdminPassword string
	DatabaseURL   string
	Addr          string
}

func Load() (Config, error) {
	cfg := Config{
		AdminPassword: os.Getenv("SHALE_ADMIN_PASSWORD"),
		DatabaseURL:   envDefault("SHALE_DATABASE_URL", DefaultDatabaseURL),
		Addr:          envDefault("SHALE_ADDR", ":8080"),
	}
	if cfg.AdminPassword == "" {
		return Config{}, errors.New("environment variable SHALE_ADMIN_PASSWORD is required and must not be empty")
	}
	return cfg, nil
}

func envDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
