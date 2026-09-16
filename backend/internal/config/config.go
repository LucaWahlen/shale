package config

import (
	"errors"
	"os"
)

type Config struct {
	AdminPassword string
	DBPath        string
	Addr          string
}

func Load() (Config, error) {
	cfg := Config{
		AdminPassword: os.Getenv("SHALE_ADMIN_PASSWORD"),
		DBPath:        envDefault("SHALE_DB_PATH", "/data/shale.db"),
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
