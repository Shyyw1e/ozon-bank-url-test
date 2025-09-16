package config

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
)

type Config struct {
	HTTPAddr		string
	LogLevel		string
	StorageBackend	string
}

func Load() (*Config, error){
	var cfg Config

	flag.StringVar(&cfg.HTTPAddr, "http-addr", getenv("HTTP_ADDR", ":8080"), "HTTP listen address")
	flag.StringVar(&cfg.LogLevel, "log-level", getenv("LOG_LEVEL", "INFO"), "log level: debug|info|warn|error")
	flag.StringVar(&cfg.StorageBackend, "storage", getenv("STORAGE_BACKEND", "memory"), "storage backend: memory|postgres")

	flag.Parse()
	switch cfg.LogLevel {
	case slog.LevelDebug.String(),
		slog.LevelInfo.String(),
		slog.LevelWarn.String(),
		slog.LevelError.String():
	default:
		return nil, fmt.Errorf("invalid log level: %s", cfg.LogLevel)
	}


	return &cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}