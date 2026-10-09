package config

import (
	"log/slog"
	"os"
	"strings"
)

// Config holds process-level settings loaded from the environment.
type Config struct {
	Port        string
	DatabaseURL string
	LogLevel    string
	Env         string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		LogLevel:    logLevel,
		Env:         env,
	}
}

// NewLogger builds a JSON slog handler at the configured level.
func NewLogger(level string) *slog.Logger {
	var lvl slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn", "warning":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(handler)
}
