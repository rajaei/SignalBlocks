package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	// Prometheus metrics removed per user request
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"signalblocks/internal/config"
	"signalblocks/internal/logging"
	"signalblocks/internal/sessionmanagement"
)

// MetricsCollector holds all application metrics
// Prometheus metrics removed

// AppContainer holds all application dependencies
type AppContainer struct {
	config              *config.Config
	logger              *logging.Logger
	// metrics removed
	redisClient         *redis.Client
	sessionRulesConfig  *sessionmanagement.SessionRulesConfig
	sessionRulesStore   *sessionmanagement.RedisSessionRulesStore
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
	httpServer          *http.Server
	uiServer            *http.Server
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger (NATS connection nil for now, will be added later)
	logger := logging.NewLogger(cfg, nil)
	defer logger.Close()

	log.Info().
		Str("version", "0.1.0-pre").
		Str("author", "Masoud Rajaeei").
		Str("environment", cfg.Environment).
		Msg("Starting SignalBlocks - Real-time Industrial Analytics Accelerator")

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		DB:           cfg.RedisDB,
		Password:     cfg.RedisPassword,
		PoolSize:     cfg.RedisMaxConnPool,
		MaxRetries:   3,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error().Err(err).Msg("Failed to connect to Redis")
		os.Exit(1)
	}
	log.Info().Msg("Connected to Redis successfully")

	// Load session rules configuration from YAML
	sessionRulesPath := "config/session-rules.yaml"
	sessionRulesConfig, err := sessionmanagement.LoadSessionRulesConfig(sessionRulesPath)
	if err != nil {
		log.Error().Err(err).Str("path", sessionRulesPath).Msg("Failed to load session rules config")
		os.Exit(1)
	}
	log.Info().
		Str("version", sessionRulesConfig.Version).
		Int("session_types", len(sessionRulesConfig.SessionTypes)).
		Msg("Session rules config loaded successfully")

	// Store session rules in Redis
	sessionRulesStore := sessionmanagement.NewRedisSessionRulesStore(redisClient)
	if err := sessionRulesStore.StoreSessionRulesConfig(ctx, sessionRulesConfig); err != nil {
		log.Error().Err(err).Msg("Failed to store session rules config in Redis")
		os.Exit(1)
	}
	log.Info().Msg("Session rules config stored in Redis successfully")

	// Create application container
	app := &AppContainer{
		config:             cfg,
		logger:             logger,
		metrics:            metrics,
		redisClient:        redisClient,
		sessionRulesConfig: sessionRulesConfig,
		sessionRulesStore:  sessionRulesStore,
		ctx:                ctx,
		cancel:             cancel,
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start services (Prometheus disabled)
	app.wg.Add(2)
	go startHTTPServer(app)
	go startMonitoringUIServer(app)

	// TODO: Initialize services
	// - NATS JetStream connection
	// - Ingest service
	// - Processing service
	// - Session builder
	// - Metadata service

	log.Info().Msg("SignalBlocks application started successfully")

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Shutdown signal received")
		app.Shutdown()
	case <-ctx.Done():
		log.Info().Msg("Context cancelled, shutting down")
		app.Shutdown()
	}
}

// Prometheus server and metrics removed

// startHTTPServer starts the HTTP API server
func startHTTPServer(app *AppContainer) {
	defer app.wg.Done()

	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Status endpoint
	mux.HandleFunc("/status", statusHandler)

	// Ingest endpoints (placeholder)
	mux.HandleFunc("/api/v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		fmt.Fprintf(w, `{"status":"accepted"}`)
	})

	// Query endpoints (placeholder)
	mux.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ready"}`)
	})

	app.httpServer = &http.Server{
		Addr:         ":" + app.config.HTTPPort,
		Handler:      mux,
		ReadTimeout:  app.config.HTTPReadTimeout,
		WriteTimeout: app.config.HTTPWriteTimeout,
	}

	log.Info().Str("port", app.config.HTTPPort).Msg("Starting HTTP API server")
	if err := app.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("HTTP server error")
	}
}

// startMonitoringUIServer starts the Monitoring UI server
func startMonitoringUIServer(app *AppContainer) {
	defer app.wg.Done()

	mux := http.NewServeMux()

	// Serve monitoring dashboard
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, dashboardHTML)
	})

	// API endpoints for real-time updates
	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"ready":true,"version":"0.1.0-pre"}`)
	})

	mux.HandleFunc("/api/metrics", metricsAPIHandler)

	app.uiServer = &http.Server{
		Addr:         ":" + app.config.MonitoringUIPort,
		Handler:      mux,
		ReadTimeout:  app.config.HTTPReadTimeout,
		WriteTimeout: app.config.HTTPWriteTimeout,
	}

	log.Info().Str("port", app.config.MonitoringUIPort).Msg("Starting Monitoring UI server")
	if err := app.uiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("Monitoring UI server error")
	}
}

// statusHandler returns service health status
func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"service":"signalblocks",
		"version":"0.1.0-pre",
		"status":"healthy",
		"uptime_seconds":%d,
		"timestamp":"%s"
	}`, time.Now().Unix(), time.Now().Format(time.RFC3339))
}

// metricsAPIHandler returns current metrics as JSON
func metricsAPIHandler(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{
		"goroutines":%d,
		"memory_alloc_bytes":%d,
		"memory_total_bytes":%d,
		"timestamp":"%s"
	}`, runtime.NumGoroutine(), m.Alloc, m.TotalAlloc, time.Now().Format(time.RFC3339))
}

// Shutdown gracefully shuts down all services
func (app *AppContainer) Shutdown() {
	log.Info().Msg("Starting graceful shutdown")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), app.config.HTTPShutdownTimeout)
	defer cancel()

	var wg sync.WaitGroup

	// Shutdown HTTP servers
	if app.httpServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := app.httpServer.Shutdown(shutdownCtx); err != nil {
				log.Error().Err(err).Msg("HTTP server shutdown error")
			}
		}()
	}

	if app.uiServer != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := app.uiServer.Shutdown(shutdownCtx); err != nil {
				log.Error().Err(err).Msg("UI server shutdown error")
			}
		}()
	}

	wg.Wait()

	// Close Redis connection
	if app.redisClient != nil {
		if err := app.redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("Redis shutdown error")
		}
	}

	// TODO: Shutdown services
	// 1. Stop accepting new connections
	// 2. Finish processing in-flight messages
	// 3. Flush queued data to storage
	// 4. Close database connections
	// 5. Close event bus connections

	app.logger.Close()

	log.Info().Msg("SignalBlocks application shut down successfully")
	os.Exit(0)
}

// dashboardHTML is the inline HTML for the monitoring dashboard
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>SignalBlocks Monitoring</title>
	<script src="https://unpkg.com/htmx.org@1.9.10"></script>
	<script src="https://cdn.tailwindcss.com"></script>
	<script src="https://unpkg.com/alpinejs@3.x.x/dist/cdn.min.js" defer></script>
	<style>
		body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
		.metric-card { @apply bg-white rounded-lg shadow p-6; }
		.status-badge { @apply inline-block px-3 py-1 rounded text-sm font-semibold; }
		.status-healthy { @apply bg-green-100 text-green-800; }
		.status-degraded { @apply bg-yellow-100 text-yellow-800; }
		.status-unhealthy { @apply bg-red-100 text-red-800; }
	</style>
</head>
<body class="bg-gray-50">
	<nav class="bg-white shadow">
		<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-4">
			<h1 class="text-2xl font-bold text-gray-900">SignalBlocks Monitoring</h1>
			<p class="text-sm text-gray-500">Real-time Industrial Analytics Accelerator</p>
		</div>
	</nav>

	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8" x-data="monitoring()">
		<!-- Status Overview -->
		<div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8">
			<div class="metric-card">
				<h3 class="text-sm font-medium text-gray-500 mb-2">Status</h3>
				<div class="flex items-center justify-between">
					<span x-text="status.status" class="text-2xl font-bold text-green-600"></span>
					<span class="status-badge status-healthy">Healthy</span>
				</div>
			</div>
			<div class="metric-card">
				<h3 class="text-sm font-medium text-gray-500 mb-2">Uptime</h3>
				<p x-text="uptime" class="text-2xl font-bold text-blue-600"></p>
			</div>
			<div class="metric-card">
				<h3 class="text-sm font-medium text-gray-500 mb-2">Goroutines</h3>
				<p x-text="metrics.goroutines" class="text-2xl font-bold text-purple-600"></p>
			</div>
			<div class="metric-card">
				<h3 class="text-sm font-medium text-gray-500 mb-2">Memory</h3>
				<p x-text="formatBytes(metrics.memory_alloc_bytes)" class="text-2xl font-bold text-indigo-600"></p>
			</div>
		</div>

		<!-- Service Status -->
		<div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
			<div class="metric-card">
				<h3 class="text-lg font-semibold mb-3">NATS JetStream</h3>
				<span class="status-badge status-healthy">Ready</span>
			</div>
			<div class="metric-card">
				<h3 class="text-lg font-semibold mb-3">Redis</h3>
				<span class="status-badge status-healthy">Ready</span>
			</div>
			<div class="metric-card">
				<h3 class="text-lg font-semibold mb-3">MinIO</h3>
				<span class="status-badge status-healthy">Ready</span>
			</div>
		</div>

		<!-- Activity Metrics -->
		<div class="grid grid-cols-1 md:grid-cols-2 gap-8">
			<div class="metric-card">
				<h3 class="text-lg font-semibold mb-4">Ingest Activity</h3>
				<div class="space-y-3">
					<div class="flex justify-between">
						<span class="text-gray-600">Messages/sec</span>
						<span class="font-semibold">0</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-600">Avg Latency</span>
						<span class="font-semibold">0ms</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-600">Error Rate</span>
						<span class="font-semibold">0%</span>
					</div>
				</div>
			</div>

			<div class="metric-card">
				<h3 class="text-lg font-semibold mb-4">Processing</h3>
				<div class="space-y-3">
					<div class="flex justify-between">
						<span class="text-gray-600">Batches/min</span>
						<span class="font-semibold">0</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-600">Avg Duration</span>
						<span class="font-semibold">0ms</span>
					</div>
					<div class="flex justify-between">
						<span class="text-gray-600">Queue Depth</span>
						<span class="font-semibold">0</span>
					</div>
				</div>
			</div>
		</div>
	</div>

	<script>
		function monitoring() {
			return {
				status: { status: 'Operational' },
				metrics: { goroutines: 0, memory_alloc_bytes: 0 },
				uptime: '0s',
				startTime: new Date(),
				formatBytes(bytes) {
					if (bytes === 0) return '0 B';
					const k = 1024;
					const sizes = ['B', 'KB', 'MB', 'GB'];
					const i = Math.floor(Math.log(bytes) / Math.log(k));
					return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
				},
				updateMetrics() {
					fetch('/api/metrics')
						.then(r => r.json())
						.then(data => {
							this.metrics = data;
							const now = new Date();
							const uptime = Math.floor((now - this.startTime) / 1000);
							this.uptime = this.formatUptime(uptime);
						})
						.catch(e => console.error('Failed to fetch metrics:', e));
				},
				formatUptime(seconds) {
					const hours = Math.floor(seconds / 3600);
					const minutes = Math.floor((seconds % 3600) / 60);
					const secs = seconds % 60;
					if (hours > 0) return hours + 'h ' + minutes + 'm';
					if (minutes > 0) return minutes + 'm ' + secs + 's';
					return secs + 's';
				},
				init() {
					this.updateMetrics();
					setInterval(() => this.updateMetrics(), 5000);
				}
			}
		}
	</script>
</body>
</html>`
