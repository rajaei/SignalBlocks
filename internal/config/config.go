package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration from environment variables
type Config struct {
	// Application
	Environment string
	LogLevel    string

	// HTTP Server
	HTTPPort             string
	HTTPReadTimeout      time.Duration
	HTTPWriteTimeout     time.Duration
	HTTPShutdownTimeout  time.Duration

	// Monitoring UI
	MonitoringUIPort           string
	MonitoringUIRefreshInterval time.Duration

	// Prometheus Metrics
	PrometheusPort  string
	PrometheusEnabled bool
	MetricsRetention time.Duration

	// NATS JetStream
	NatsURL                   string
	NatsMaxReconnectAttempts  int
	NatsReconnectWait         time.Duration
	NatsRequestTimeout        time.Duration

	// Redis
	RedisAddr          string
	RedisDB            int
	RedisPassword      string
	RedisMaxConnPool   int
	RedisIdleTimeout   time.Duration

	// MinIO S3
	MinioEndpoint      string
	MinioAccessKey     string
	MinioSecretKey     string
	MinioUseSsl        bool
	MinioRegion        string
	MinioBucketData    string
	MinioPartitionLayout string

	// Processing
	IngestWorkers              int
	ProcessingWorkers          int
	IngestBatchSize            int
	IngestBatchTimeout         time.Duration
	ProcessingBatchSize        int
	ProcessingBatchTimeout     time.Duration

	// Session Builder
	SessionBuilderEnabled      bool
	SessionChangeCheckInterval time.Duration
	SessionFlushInterval       time.Duration

	// Parquet
	ParquetPageSize      int
	ParquetRowGroupSize  int
	ParquetCompression   string
	ParquetWriteTimeout  time.Duration

	// Tag Configuration
	TagCacheTTL           time.Duration
	TagMaxCacheSize       int
	TagMetadataUpdateInterval time.Duration

	// Logging
	ZerologLevel       string
	LokiEnabled        bool
	LokiURL            string
	LokiBatchSize      int
	LokiBatchTimeout   time.Duration

	// Grafana
	GrafanaPassword    string
	GrafanaProvisioning bool

	// SLA Targets
	QueryLatencyTargetMs    float64
	IngestThroughputTargetPerSec int

	// Feature Flags
	FeatureMultiNodeReady   bool
	FeatureDuckDBQueryEngine bool
	FeatureRedisLiveIndex   bool
	FeatureSessionTracking bool

	// Debug
	DebugMode           bool
	DebugTraceIngest    bool
	DebugTraceProcessing bool
	DebugTraceStorage   bool
	MockMQTTPublisher   bool
	MockDataRate        int
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		// Application
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "debug"),

		// HTTP Server
		HTTPPort:            getEnv("HTTP_PORT", "8080"),
		HTTPReadTimeout:     getDuration("HTTP_READ_TIMEOUT_SEC", 30*time.Second),
		HTTPWriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT_SEC", 30*time.Second),
		HTTPShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT_SEC", 10*time.Second),

		// Monitoring UI
		MonitoringUIPort:            getEnv("MONITORING_UI_PORT", "8081"),
		MonitoringUIRefreshInterval: getDurationMs("MONITORING_UI_REFRESH_INTERVAL_MS", 5000*time.Millisecond),

		// Prometheus Metrics
		PrometheusPort:    getEnv("PROMETHEUS_PORT", "9091"),
		PrometheusEnabled: getBool("PROMETHEUS_ENABLED", false),
		MetricsRetention: getDuration("METRICS_RETENTION_HOURS", 24*time.Hour),

		// NATS JetStream
		NatsURL:                  getEnv("NATS_URL", "nats://localhost:4222"),
		NatsMaxReconnectAttempts: getInt("NATS_MAX_RECONNECT_ATTEMPTS", 10),
		NatsReconnectWait:        getDurationMs("NATS_RECONNECT_WAIT_MS", 250*time.Millisecond),
		NatsRequestTimeout:       getDurationMs("NATS_REQUEST_TIMEOUT_MS", 5000*time.Millisecond),

		// Redis
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		RedisDB:          getInt("REDIS_DB", 0),
		RedisPassword:    getEnv("REDIS_PASSWORD", ""),
		RedisMaxConnPool: getInt("REDIS_MAX_CONN_POOL", 100),
		RedisIdleTimeout: getDuration("REDIS_IDLE_TIMEOUT_SEC", 300*time.Second),

		// MinIO S3
		MinioEndpoint:       getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey:      getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinioSecretKey:      getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinioUseSsl:         getBool("MINIO_USE_SSL", false),
		MinioRegion:         getEnv("MINIO_REGION", "us-east-1"),
		MinioBucketData:     getEnv("MINIO_BUCKET_DATA", "signalblocks-data"),
		MinioPartitionLayout: getEnv("MINIO_PARTITION_LAYOUT", "year/month/day/group"),

		// Processing
		IngestWorkers:        getInt("INGEST_WORKERS", 4),
		ProcessingWorkers:    getInt("PROCESSING_WORKERS", 4),
		IngestBatchSize:      getInt("INGEST_BATCH_SIZE", 1000),
		IngestBatchTimeout:   getDurationMs("INGEST_BATCH_TIMEOUT_MS", 100*time.Millisecond),
		ProcessingBatchSize:  getInt("PROCESSING_BATCH_SIZE", 10000),
		ProcessingBatchTimeout: getDurationMs("PROCESSING_BATCH_TIMEOUT_MS", 500*time.Millisecond),

		// Session Builder
		SessionBuilderEnabled:      getBool("SESSION_BUILDER_ENABLED", true),
		SessionChangeCheckInterval: getDurationMs("SESSION_CHANGE_CHECK_INTERVAL_MS", 1000*time.Millisecond),
		SessionFlushInterval:       getDurationMs("SESSION_FLUSH_INTERVAL_MS", 5000*time.Millisecond),

		// Parquet
		ParquetPageSize:     getInt("PARQUET_PAGE_SIZE", 1048576),
		ParquetRowGroupSize: getInt("PARQUET_ROW_GROUP_SIZE", 131072),
		ParquetCompression:  getEnv("PARQUET_COMPRESSION", "SNAPPY"),
		ParquetWriteTimeout: getDuration("PARQUET_WRITE_TIMEOUT_SEC", 60*time.Second),

		// Tag Configuration
		TagCacheTTL:                getDuration("TAG_CACHE_TTL_SEC", 3600*time.Second),
		TagMaxCacheSize:            getInt("TAG_MAX_CACHE_SIZE", 100000),
		TagMetadataUpdateInterval:  getDurationMs("TAG_METADATA_UPDATE_INTERVAL_MS", 5000*time.Millisecond),

		// Logging
		ZerologLevel:     getEnv("ZEROLOG_LEVEL", "debug"),
		LokiEnabled:      getBool("LOKI_ENABLED", false),
		LokiURL:          getEnv("LOKI_URL", "http://localhost:3100"),
		LokiBatchSize:    getInt("LOKI_BATCH_SIZE", 100),
		LokiBatchTimeout: getDurationMs("LOKI_BATCH_TIMEOUT_MS", 1000*time.Millisecond),

		// Grafana
		GrafanaPassword:    getEnv("GRAFANA_PASSWORD", "admin"),
		GrafanaProvisioning: getBool("GRAFANA_PROVISIONING_ENABLED", true),

		// SLA Targets
		QueryLatencyTargetMs:    getFloat("QUERY_LATENCY_TARGET_MS", 2.0),
		IngestThroughputTargetPerSec: getInt("INGEST_THROUGHPUT_TARGET_MSGS_SEC", 100000),

		// Feature Flags
		FeatureMultiNodeReady:       getBool("FEATURE_MULTI_NODE_READY", false),
		FeatureDuckDBQueryEngine:    getBool("FEATURE_DUCKDB_QUERY_ENGINE", true),
		FeatureRedisLiveIndex:       getBool("FEATURE_REDIS_LIVE_INDEX", true),
		FeatureSessionTracking:      getBool("FEATURE_SESSION_TRACKING", true),

		// Debug
		DebugMode:            getBool("DEBUG_MODE", false),
		DebugTraceIngest:     getBool("DEBUG_TRACE_INGEST", false),
		DebugTraceProcessing: getBool("DEBUG_TRACE_PROCESSING", false),
		DebugTraceStorage:    getBool("DEBUG_TRACE_STORAGE", false),
		MockMQTTPublisher:    getBool("MOCK_MQTT_PUBLISHER", false),
		MockDataRate:         getInt("MOCK_DATA_RATE_HZ", 100),
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true" || value == "1" || strings.ToLower(value) == "yes"
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return defaultValue
}

func getDurationMs(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if ms, err := strconv.ParseInt(value, 10, 64); err == nil {
			return time.Duration(ms) * time.Millisecond
		}
	}
	return defaultValue
}
