package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	AppName       string
	Env           string
	Addr          string
	DataDir       string
	DBPath        string
	AdminPassword string
	LogLevel      slog.Level
	AutoRetryTick time.Duration
}

func MustLoadConfig() Config {
	dataDir := envOrDefault("CLOUDPAN_DATA_DIR", filepath.Join(".", ".cloudpan-sync-go"))
	return Config{
		AppName:       envOrDefault("CLOUDPAN_APP_NAME", "CloudPan Sync Go"),
		Env:           envOrDefault("CLOUDPAN_ENV", "development"),
		Addr:          envOrDefault("CLOUDPAN_ADDR", ":8080"),
		DataDir:       dataDir,
		DBPath:        envOrDefault("CLOUDPAN_DB_PATH", filepath.Join(dataDir, "cloudpan-sync.db")),
		AdminPassword: envOrDefault("CLOUDPAN_ADMIN_PASSWORD", "admin"),
		LogLevel:      parseLogLevel(envOrDefault("CLOUDPAN_LOG_LEVEL", "info")),
		AutoRetryTick: parseDurationOrDefault("CLOUDPAN_AUTO_RETRY_TICK", 3*time.Second),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseLogLevel(value string) slog.Level {
	switch value {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func parseDurationOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	if duration < 0 {
		return fallback
	}
	return duration
}
