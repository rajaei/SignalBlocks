package types

import (
	"encoding/json"
	"time"
)

// RawDataPoint represents a single raw measurement from a sensor/tag
type RawDataPoint struct {
	// Identification
	TagID       string    `json:"tag_id"`       // Unique tag identifier (e.g., "furnace.temp.1")
	GroupID     string    `json:"group_id"`     // Group identifier (e.g., "furnace_group_001")
	Timestamp   time.Time `json:"timestamp"`    // UTC timestamp
	ReceivedAt  time.Time `json:"received_at"`  // When received by ingest

	// Data
	Value   float64 `json:"value"`   // Raw numeric value (e.g., temperature in Celsius)
	Quality int8    `json:"quality"` // Data quality: 0=good, 1=questionable, 2=bad, -1=null

	// Metadata
	SourceID     string `json:"source_id"`                // Source identifier (MQTT topic, WebSocket client, etc.)
	Sequence     uint64 `json:"sequence,omitempty"`       // Optional sequence number for ordering
	BatchID      string `json:"batch_id,omitempty"`       // Batch identifier for grouping
	CustomFields map[string]interface{} `json:"custom_fields,omitempty"` // For extensibility
}

// TagMetadata represents tag configuration and metadata (stored in Redis)
type TagMetadata struct {
	TagID        string                 `redis:"tag_id"`        // Primary key
	Name         string                 `redis:"name"`          // Human-readable name
	Unit         string                 `redis:"unit"`          // Unit of measurement (°C, kW, etc.)
	GroupID      string                 `redis:"group_id"`      // Associated group
	DataType     string                 `redis:"data_type"`     // "float64", "int32", etc.
	ChangePolicy string                 `redis:"change_policy"` // "on_change", "periodic", "always"
	Category     string                 `redis:"category"`      // "temperature", "pressure", "power", etc.
	Min          float64                `redis:"min"`           // Minimum expected value
	Max          float64                `redis:"max"`           // Maximum expected value
	Enabled      bool                   `redis:"enabled"`       // Whether tag is actively monitored
	TagIndex     int16                  `redis:"tag_index"`     // Index for Parquet encoding
	CreatedAt    int64                  `redis:"created_at"`    // Unix timestamp
	UpdatedAt    int64                  `redis:"updated_at"`    // Unix timestamp
	CustomMeta   map[string]string      `redis:"custom_meta"`   // Custom metadata
}

// StateVector represents the aggregated state of all tags in a segment
// Used for bit-packed storage of tag change states
type StateVector struct {
	SegmentStart time.Time      `parquet:"segment_start"` // Start of time segment
	SegmentEnd   time.Time      `parquet:"segment_end"`   // End of time segment
	Vector       []byte         `parquet:"vector"`        // Bit-packed state (2 bits per tag)
	ChangeCount  uint32         `parquet:"change_count"`  // Number of changes in segment
	GroupID      string         `parquet:"group_id"`      // Group identifier
	PartitionKey string         `parquet:"partition_key"` // For Hive partitioning: year/month/day/group
}

// ValueData represents individual tag values (stored in value_data.parquet)
type ValueData struct {
	SegmentStart time.Time `parquet:"segment_start"` // Start of segment
	TagIndex     int16     `parquet:"tag_index"`     // Reference to tag metadata (Redis lookup)
	Timestamp    time.Time `parquet:"timestamp"`     // Exact timestamp
	Value        float64   `parquet:"value"`         // Numeric value
	Quality      int8      `parquet:"quality"`       // Data quality flag
	GroupID      string    `parquet:"group_id"`      // Group identifier
	PartitionKey string    `parquet:"partition_key"` // For Hive partitioning: year/month/day/group
}

// ProcessingBatch represents a batch of raw data points ready for processing
type ProcessingBatch struct {
	BatchID      string         `json:"batch_id"`
	GroupID      string         `json:"group_id"`
	DataPoints   []RawDataPoint `json:"data_points"`
	BatchSize    int            `json:"batch_size"`
	ProcessedAt  time.Time      `json:"processed_at"`
	SegmentStart time.Time      `json:"segment_start"`
	SegmentEnd   time.Time      `json:"segment_end"`
	ChangeCount  uint32         `json:"change_count"`
	QualityScore float64        `json:"quality_score"` // 0-100, percentage of good quality data
}

// SessionChangeEvent represents a session state change event (generic, triggered by configured tag)
type SessionChangeEvent struct {
	SessionID    string                 `json:"session_id"`
	SessionType  string                 `json:"session_type"`  // "heat", "shift", "campaign"
	GroupID      string                 `json:"group_id"`
	EventType    string                 `json:"event_type"`    // "started", "ended", "paused", "resumed"
	TriggerTag   string                 `json:"trigger_tag"`   // Which tag triggered this event
	TriggerValue interface{}            `json:"trigger_value"` // The value of the trigger tag
	Timestamp    time.Time              `json:"timestamp"`
	Reason       string                 `json:"reason,omitempty"`
	OperatorID   string                 `json:"operator_id,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// QueryRequest represents a query against Parquet data via DuckDB
type QueryRequest struct {
	QueryID      string        `json:"query_id"`
	GroupID      string        `json:"group_id"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	TagIDs       []string      `json:"tag_ids"`       // Empty = all tags in group
	Aggregation  string        `json:"aggregation"`   // "none", "1m", "5m", "1h", etc.
	Where        string        `json:"where,omitempty"` // SQL WHERE clause conditions
	Limit        int           `json:"limit,omitempty"`
	CustomSQL    string        `json:"custom_sql,omitempty"` // Raw DuckDB SQL (if allowed)
	SubmittedAt  time.Time     `json:"submitted_at"`
}

// QueryResult represents the result of a query
type QueryResult struct {
	QueryID      string                   `json:"query_id"`
	Status       string                   `json:"status"`       // "success", "error", "timeout"
	Rows         []map[string]interface{} `json:"rows"`
	RowCount     int                      `json:"row_count"`
	ExecutionMs  float64                  `json:"execution_ms"`
	Error        string                   `json:"error,omitempty"`
	CompletedAt  time.Time                `json:"completed_at"`
}

// HealthStatus represents service health information
type HealthStatus struct {
	Service   string                 `json:"service"`
	Status    string                 `json:"status"`     // "healthy", "degraded", "unhealthy"
	Uptime    int64                  `json:"uptime_sec"`
	Details   map[string]interface{} `json:"details"`
	CheckedAt time.Time              `json:"checked_at"`
}

// SystemMetrics represents current system-wide metrics
type SystemMetrics struct {
	Timestamp              time.Time `json:"timestamp"`
	MessagesIngestedTotal  int64     `json:"messages_ingested_total"`
	MessagesProcessedTotal int64     `json:"messages_processed_total"`
	ActiveSessions         int       `json:"active_sessions"`
	AvgIngestLatencyMs     float64   `json:"avg_ingest_latency_ms"`
	AvgQueryLatencyMs      float64   `json:"avg_query_latency_ms"`
	StorageUsedBytes       int64     `json:"storage_used_bytes"`
	GoroutinesCount        int       `json:"goroutines_count"`
	MemoryUsageBytes       int64     `json:"memory_usage_bytes"`
}

// EventBusMessage is the envelope for all NATS JetStream messages
type EventBusMessage struct {
	Type        string                 `json:"type"`        // "raw_data", "batch_processed", "session_changed", "log"
	Subject     string                 `json:"subject"`     // NATS subject
	Timestamp   time.Time              `json:"timestamp"`
	SourceID    string                 `json:"source_id"`
	Payload     json.RawMessage        `json:"payload"`
	Metadata    map[string]string      `json:"metadata"`
	CorrelationID string               `json:"correlation_id"` // For tracing
}

// LogEntry represents a structured log entry published to NATS logs.* subject
type LogEntry struct {
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`         // "debug", "info", "warn", "error"
	Service       string                 `json:"service"`
	Component     string                 `json:"component"`
	Message       string                 `json:"message"`
	Error         string                 `json:"error,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	TraceID       string                 `json:"trace_id,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
}
