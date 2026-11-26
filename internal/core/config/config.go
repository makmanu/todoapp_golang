package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddress      string
	ShutdownTimeout  time.Duration
	PostgresURL      string
	PostgresTimeout  time.Duration
	PostgresMaxConns int32
}

func NewConfigMust() Config {
	cfg, err := NewConfig()
	if err != nil {
		panic(err)
	}
	return cfg
}
func NewConfig() (Config, error) {
	var cfg Config
	cfg.HTTPAddress = env("HTTP_ADDR", "127.0.0.1:8080")
	if _, _, err := net.SplitHostPort(cfg.HTTPAddress); err != nil {
		return cfg, fmt.Errorf("HTTP_ADDR: %w", err)
	}
	var err error
	cfg.ShutdownTimeout, err = duration("HTTP_SHUTDOWN_TIMEOUT", "10s")
	if err != nil {
		return cfg, err
	}
	cfg.PostgresTimeout, err = duration("POSTGRES_TIMEOUT", "10s")
	if err != nil {
		return cfg, err
	}
	n, err := strconv.ParseInt(env("POSTGRES_MAX_CONNS", "10"), 10, 32)
	if err != nil || n < 1 {
		return cfg, fmt.Errorf("POSTGRES_MAX_CONNS must be a positive 32-bit integer")
	}
	cfg.PostgresMaxConns = int32(n)
	for _, name := range []string{"POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"} {
		if os.Getenv(name) == "" {
			return cfg, fmt.Errorf("%s is required", name)
		}
	}
	port := env("POSTGRES_PORT", "5432")
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return cfg, fmt.Errorf("POSTGRES_PORT must be between 1 and 65535")
	}
	sslmode := env("POSTGRES_SSLMODE", "disable")
	switch sslmode {
	case "disable", "allow", "prefer", "require", "verify-ca", "verify-full":
	default:
		return cfg, fmt.Errorf("invalid POSTGRES_SSLMODE")
	}
	u := url.URL{Scheme: "postgres", User: url.UserPassword(os.Getenv("POSTGRES_USER"), os.Getenv("POSTGRES_PASSWORD")),
		Host: net.JoinHostPort(env("POSTGRES_HOST", "localhost"), port), Path: "/" + os.Getenv("POSTGRES_DB")}
	q := u.Query()
	q.Set("sslmode", sslmode)
	q.Set("search_path", "public")
	u.RawQuery = q.Encode()
	cfg.PostgresURL = u.String()
	return cfg, nil
}
func env(name, fallback string) string {
	if s := os.Getenv(name); s != "" {
		return s
	}
	return fallback
}
func duration(name, fallback string) (time.Duration, error) {
	d, err := time.ParseDuration(env(name, fallback))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("%s must be positive", name)
	}
	return d, nil
}
