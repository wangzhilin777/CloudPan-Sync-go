package app

import (
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	AppName                        string
	Env                            string
	Addr                           string
	DataDir                        string
	DBPath                         string
	AdminPassword                  string
	LogLevel                       slog.Level
	AutoRetryTick                  time.Duration
	AutoRetryBatchLimit            int
	AutoRetryLimitPerMode          int
	AutoRetryLimitPerLane          int
	AutoRetryLimitPerProtocolGroup int
	AutoRetryLimitPerProvider      int
	AutoRetryLimitPerProfile       int
}

func MustLoadConfig() Config {
	dataDir := envOrDefault("CLOUDPAN_DATA_DIR", filepath.Join(".", ".cloudpan-sync-go"))
	return Config{
		AppName:                        envOrDefault("CLOUDPAN_APP_NAME", "CloudPan Sync Go"),
		Env:                            envOrDefault("CLOUDPAN_ENV", "development"),
		Addr:                           envOrDefault("CLOUDPAN_ADDR", ":8080"),
		DataDir:                        dataDir,
		DBPath:                         envOrDefault("CLOUDPAN_DB_PATH", filepath.Join(dataDir, "cloudpan-sync.db")),
		AdminPassword:                  envOrDefault("CLOUDPAN_ADMIN_PASSWORD", "admin"),
		LogLevel:                       parseLogLevel(envOrDefault("CLOUDPAN_LOG_LEVEL", "info")),
		AutoRetryTick:                  parseDurationOrDefault("CLOUDPAN_AUTO_RETRY_TICK", 3*time.Second),
		AutoRetryBatchLimit:            parseIntOrDefault("CLOUDPAN_AUTO_RETRY_BATCH_LIMIT", 3),
		AutoRetryLimitPerMode:          parseIntOrDefault("CLOUDPAN_AUTO_RETRY_LIMIT_PER_MODE", 1),
		AutoRetryLimitPerLane:          parseIntOrDefault("CLOUDPAN_AUTO_RETRY_LIMIT_PER_LANE", 1),
		AutoRetryLimitPerProtocolGroup: parseIntOrDefault("CLOUDPAN_AUTO_RETRY_LIMIT_PER_PROTOCOL_GROUP", 1),
		AutoRetryLimitPerProvider:      parseIntOrDefault("CLOUDPAN_AUTO_RETRY_LIMIT_PER_PROVIDER", 1),
		AutoRetryLimitPerProfile:       parseIntOrDefault("CLOUDPAN_AUTO_RETRY_LIMIT_PER_PROFILE", 1),
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

func parseIntOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}
