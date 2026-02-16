package config

import (
	"os"
	"strconv"
	//"strings"
	"time"
)

// Config تمام تنظیمات برنامه
type Config struct {
	Environment string
	LogLevel    string

	NATSURL      string
	RedisAddr    string
	RedisDB      int
	RedisPassword string

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioUseSSL    bool
	MinioBucket    string

	IngestWorkers     int
	ProcessingWorkers int

	SessionBuilderEnabled bool

	// Logging
	ZerologLevel string
	LokiEnabled  bool
	LokiURL      string
	LokiBatchSize int
	LokiBatchTimeout time.Duration
}

func Load() Config {
	return Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),

		NATSURL:      getEnv("NATS_URL", "nats://nats:4222"),
		RedisAddr:    getEnv("REDIS_ADDR", "redis:6379"),
		RedisDB:      getInt("REDIS_DB", 0),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "minio:9000"),
		MinioAccessKey: getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_ROOT_PASSWORD", "minioadmin123"),
		MinioUseSSL:    getBool("MINIO_USE_SSL", false),
		MinioBucket:    getEnv("MINIO_BUCKET", "signalblocks"),

		IngestWorkers:     getInt("INGEST_WORKERS", 4),
		ProcessingWorkers: getInt("PROCESSING_WORKERS", 4),

		SessionBuilderEnabled: getBool("SESSION_BUILDER_ENABLED", true),

		ZerologLevel:     getEnv("ZEROLOG_LEVEL", "debug"),
		LokiEnabled:      getBool("LOKI_ENABLED", false),
		LokiURL:          getEnv("LOKI_URL", "http://loki:3100/loki/api/v1/push"),
		LokiBatchSize:    getInt("LOKI_BATCH_SIZE", 100),
		LokiBatchTimeout: time.Duration(getInt("LOKI_BATCH_TIMEOUT_MS", 500)) * time.Millisecond,
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getInt(key string, defaultValue int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func getBool(key string, defaultValue bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1" || v == "yes"
	}
	return defaultValue
}