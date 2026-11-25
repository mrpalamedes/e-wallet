package config

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const minSecretLen = 32

type Config struct {
	HTTPAddr             string
	DatabaseURL          string
	JWTSecret            string
	JWTTTL               time.Duration
	PaymentSessionTTL    time.Duration
	PaymentSweepInterval time.Duration
}

func Load() (Config, error) {
	jwtTTL, err := durationEnv("JWT_TTL", "24h")
	if err != nil {
		return Config{}, err
	}
	paymentTTL, err := durationEnv("PAYMENT_SESSION_TTL", "10m")
	if err != nil {
		return Config{}, err
	}
	sweepInterval, err := durationEnv("PAYMENT_SWEEP_INTERVAL", "30s")
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		HTTPAddr:             getEnv("HTTP_ADDR", ":8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://ewallet:ewallet@localhost:5432/ewallet?sslmode=disable"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		JWTTTL:               jwtTTL,
		PaymentSessionTTL:    paymentTTL,
		PaymentSweepInterval: sweepInterval,
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

func durationEnv(key, fallback string) (time.Duration, error) {
	duration, err := time.ParseDuration(getEnv(key, fallback))
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	if duration <= 0 {
		return 0, fmt.Errorf("%s must be positive", key)
	}
	return duration, nil
}
