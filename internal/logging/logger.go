package logging

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/rajaei/SignalBlocks/internal/config"
	"github.com/rajaei/SignalBlocks/pkg/types"
)

// Logger wraps zerolog with NATS integration
type Logger struct {
	zl          zerolog.Logger
	natsConn    *nats.Conn
	natsChan    chan *nats.EncodedConn
	eventQueue  chan types.LogEntry
	flushTicker *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.Mutex
}

// NewLogger creates a new logger instance with optional NATS integration
func NewLogger(cfg *config.Config, natsConn *nats.Conn) *Logger {
	initZerolog(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	logger := &Logger{
		zl:         log.With().Str("component", "logger").Logger(),
		natsConn:   natsConn,
		eventQueue: make(chan types.LogEntry, 10000),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start background event processor if NATS is available
	if natsConn != nil && cfg.LokiEnabled {
		logger.wg.Add(1)
		go logger.eventProcessor(cfg)
	}

	return logger
}

// initZerolog initializes the global Zerolog logger
func initZerolog(cfg *config.Config) {
	// Pretty print for development, JSON for production
	if cfg.Environment == "production" {
		zerolog.SetGlobalLevel(parseLogLevel(cfg.ZerologLevel))
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: "15:04:05",
		})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// parseLogLevel converts string to zerolog.Level
func parseLogLevel(level string) zerolog.Level {
	switch level {
	case "panic":
		return zerolog.PanicLevel
	case "fatal":
		return zerolog.FatalLevel
	case "error":
		return zerolog.ErrorLevel
	case "warn":
		return zerolog.WarnLevel
	case "info":
		return zerolog.InfoLevel
	case "debug":
		return zerolog.DebugLevel
	case "trace":
		return zerolog.TraceLevel
	default:
		return zerolog.InfoLevel
	}
}

// WithService returns a logger with service context
func (l *Logger) WithService(service string) zerolog.Logger {
	return l.zl.With().Str("service", service).Logger()
}

// WithComponent returns a logger with component context
func (l *Logger) WithComponent(component string) zerolog.Logger {
	return l.zl.With().Str("component", component).Logger()
}

// WithCorrelation returns a logger with correlation ID
func (l *Logger) WithCorrelation(correlationID string) zerolog.Logger {
	return l.zl.With().Str("correlation_id", correlationID).Logger()
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...interface{}) {
	l.log(zerolog.InfoLevel, msg, fields...)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.log(zerolog.DebugLevel, msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.log(zerolog.WarnLevel, msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...interface{}) {
	l.log(zerolog.ErrorLevel, msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...interface{}) {
	l.log(zerolog.FatalLevel, msg, fields...)
}

// log is the internal logging function
func (l *Logger) log(level zerolog.Level, msg string, fields ...interface{}) {
	event := l.zl.WithLevel(level)

	for i := 0; i < len(fields)-1; i += 2 {
		if key, ok := fields[i].(string); ok {
			event = event.Interface(key, fields[i+1])
		}
	}

	event.Msg(msg)

	// Also queue to NATS for log aggregation
	if l.natsConn != nil {
		select {
		case l.eventQueue <- types.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     level.String(),
			Message:   msg,
		}:
		default:
			// Queue full, skip async logging
		}
	}
}

// eventProcessor processes and sends logs to NATS logs.* subject
func (l *Logger) eventProcessor(cfg *config.Config) {
	defer l.wg.Done()

	if l.natsConn == nil {
		return
	}

	batch := make([]types.LogEntry, 0, cfg.LokiBatchSize)
	ticker := time.NewTicker(cfg.LokiBatchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-l.ctx.Done():
			// Flush any remaining logs before exit
			if len(batch) > 0 {
				l.flushLogs(batch)
			}
			return

		case logEntry := <-l.eventQueue:
			batch = append(batch, logEntry)
			if len(batch) >= cfg.LokiBatchSize {
				l.flushLogs(batch)
				batch = make([]types.LogEntry, 0, cfg.LokiBatchSize)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				l.flushLogs(batch)
				batch = make([]types.LogEntry, 0, cfg.LokiBatchSize)
			}
		}
	}
}

// flushLogs publishes a batch of logs to NATS
func (l *Logger) flushLogs(batch []types.LogEntry) {
	if l.natsConn == nil || len(batch) == 0 {
		return
	}

	for _, entry := range batch {
		subject := "logs." + entry.Level
		if entry.Service != "" {
			subject = "logs." + entry.Service
		}

		data, err := json.Marshal(entry)
		if err != nil {
			l.zl.Error().Err(err).Msg("Failed to marshal log entry")
			continue
		}

		if err := l.natsConn.Publish(subject, data); err != nil {
			l.zl.Error().Err(err).Msg("Failed to publish log to NATS")
		}
	}
}

// Close gracefully shuts down the logger
func (l *Logger) Close() error {
	l.cancel()
	l.wg.Wait()
	close(l.eventQueue)
	return nil
}

// LogHook is a hook for writing to an io.Writer and NATS
type LogHook struct {
	writer io.Writer
	logger *Logger
}

// NewLogHook creates a new log hook
func NewLogHook(writer io.Writer, logger *Logger) *LogHook {
	return &LogHook{
		writer: writer,
		logger: logger,
	}
}

// Run is called by zerolog for each log entry
func (h *LogHook) Run(e *zerolog.Event, level zerolog.Level, msg string) {
	if h.writer != nil {
		h.writer.Write([]byte(msg + "\n"))
	}
}

// ContextLogger provides request-scoped logging
type ContextLogger struct {
	base      zerolog.Logger
	traceID   string
	requestID string
}

// FromContext extracts logger from context or creates new one
func FromContext(ctx context.Context, base zerolog.Logger) *ContextLogger {
	traceID := ""
	requestID := ""

	if val := ctx.Value("trace_id"); val != nil {
		if str, ok := val.(string); ok {
			traceID = str
		}
	}

	if val := ctx.Value("request_id"); val != nil {
		if str, ok := val.(string); ok {
			requestID = str
		}
	}

	return &ContextLogger{
		base:      base,
		traceID:   traceID,
		requestID: requestID,
	}
}

// WithContext adds context to the logger
func (cl *ContextLogger) WithContext(ctx context.Context) zerolog.Logger {
	logger := cl.base
	if cl.traceID != "" {
		logger = logger.With().Str("trace_id", cl.traceID).Logger()
	}
	if cl.requestID != "" {
		logger = logger.With().Str("request_id", cl.requestID).Logger()
	}
	return logger
}
