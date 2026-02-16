package config

import (
	"encoding/json"
	//"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/rs/zerolog/log"
)

// Config تمام تنظیمات برنامه
type Config struct {
	Environment string `json:"environment"`
	LogLevel    string `json:"log_level"`

	NATSURL       string `json:"nats_url"`
	RedisAddr     string `json:"redis_addr"`
	RedisDB       int    `json:"redis_db"`
	RedisPassword string `json:"redis_password"`

	MinioEndpoint  string `json:"minio_endpoint"`
	MinioAccessKey string `json:"minio_access_key"`
	MinioSecretKey string `json:"minio_secret_key"`
	MinioUseSSL    bool   `json:"minio_use_ssl"`
	MinioBucket    string `json:"minio_bucket"`

	IngestWorkers     int `json:"ingest_workers"`
	ProcessingWorkers int `json:"processing_workers"`

	SessionBuilderEnabled bool `json:"session_builder_enabled"`

	EnableLoadBalancing bool `json:"enable_load_balancing"`
	IngestGroups        int  `json:"ingest_groups"`
	MaxTagsPerMessage   int  `json:"max_tags_per_message"`

	// Logging
	ZerologLevel     string `json:"zerolog_level"`
	LokiEnabled      bool   `json:"loki_enabled"`
	LokiURL          string `json:"loki_url"`
	LokiBatchSize    int    `json:"loki_batch_size"`
	LokiBatchTimeout int    `json:"loki_batch_timeout"` // میلی‌ثانیه
}

// ConfigManager مدیریت کانفیگ
type ConfigManager struct {
	config Config
	mu     sync.RWMutex
}

// NewConfigManager - اولویت با فایل JSON، fallback به env
func NewConfigManager(filePath string) *ConfigManager {
	var cfg Config

	// اول سعی کن از فایل بخونه
	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err == nil {
			if err := json.Unmarshal(data, &cfg); err == nil {
				log.Info().Str("file", filePath).Msg("Config loaded from JSON")
				return &ConfigManager{config: cfg}
			}
			log.Warn().Err(err).Msg("Invalid config.json - falling back to env")
		} else {
			log.Warn().Err(err).Str("file", filePath).Msg("Config file not found - falling back to env")
		}
	}

	// fallback به env
	cfg = Config{
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),

		NATSURL:       getEnv("NATS_URL", "nats://172.21.0.4:4222"),
		RedisAddr:     getEnv("REDIS_ADDR", "172.21.0.4:6379"),
		RedisDB:       getInt("REDIS_DB", 0),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "172.21.0.4:9000"),
		MinioAccessKey: getEnv("MINIO_ROOT_USER", "minioadmin"),
		MinioSecretKey: getEnv("MINIO_ROOT_PASSWORD", "minioadmin123"),
		MinioUseSSL:    getBool("MINIO_USE_SSL", false),
		MinioBucket:    getEnv("MINIO_BUCKET", "signalblocks"),

		IngestWorkers:     getInt("INGEST_WORKERS", 4),
		ProcessingWorkers: getInt("PROCESSING_WORKERS", 4),

		SessionBuilderEnabled: getBool("SESSION_BUILDER_ENABLED", true),

		EnableLoadBalancing: getBool("ENABLE_LOAD_BALANCING", true),
		IngestGroups:        getInt("INGEST_GROUPS", 4),
		MaxTagsPerMessage:   getInt("MAX_TAGS_PER_MESSAGE", 100),

		ZerologLevel:     getEnv("ZEROLOG_LEVEL", "debug"),
		LokiEnabled:      getBool("LOKI_ENABLED", false),
		LokiURL:          getEnv("LOKI_URL", "http://172.21.0.4:3100/loki/api/v1/push"),
		LokiBatchSize:    getInt("LOKI_BATCH_SIZE", 100),
		LokiBatchTimeout: getInt("LOKI_BATCH_TIMEOUT_MS", 500),
	}

	return &ConfigManager{config: cfg}
}

// GetConfig - کپی امن کانفیگ
func (cm *ConfigManager) GetConfig() Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// Helper functions
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
