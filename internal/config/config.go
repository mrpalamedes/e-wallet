package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const minSecretLen = 32

type Config struct {
	HTTPAddr    string
	DatabaseUrl string
	JWTSecret   string
	JWTTTL      time.Duration
}

func Load() (Config, error) {
	ttl, err := time.ParseDuration(getEnv("JWT_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_TTL: %w", err)
	}

	cfg := Config{
		HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
		DatabaseUrl: getEnv("DATABASE_URL", "postgres://ewallet:ewallet@localhost:5432/ewallet?sslmode=disable"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		JWTTTL:      ttl,
	}

	if len(cfg.JWTSecret) < minSecretLen {
		return Config{}, errors.New("JWT_SECRET is required and must be at least 32 characters")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
