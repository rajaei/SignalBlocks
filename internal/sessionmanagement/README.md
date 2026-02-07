# Session Management System

SignalBlocks includes a fully configurable session management system that dynamically calculates KPIs based on YAML configuration.

## Overview

Sessions (heat, shift, campaign) are tracked based on configurable rules that determine:
- When a new session starts (trigger condition)
- What KPIs to calculate (type, aggregation method, conditions)
- Where to store session data (Parquet file and partitioning)
- How to publish updates (Redis pub/sub)

## Configuration File

The session rules are defined in `config/session-rules.yaml`:

```yaml
session_types:
  heat:
    enabled: true
    trigger:
      tag_name: "heat_number"      # Which tag identifies the session
      change_policy: "on_change"   # When to trigger a new session
    
    kpis:
      - name: "duration"
        type: "duration"           # How to calculate (duration, sum, avg, max, min, count)
        output_column: "duration_seconds"
      
      - name: "total_energy"
        type: "sum"
        source_tag: "energy_kwh"   # Which tag to aggregate
        output_column: "total_energy_kwh"
        condition: "value > 0"     # Optional condition filter
```

## Architecture

### 1. Configuration Loading (`sessionmanagement/config.go`)

At startup, the YAML file is loaded and validated:

```go
config, err := sessionmanagement.LoadSessionRulesConfig("config/session-rules.yaml")
if err != nil {
    log.Fatal(err)
}
```

Validation checks:
- All required fields are present
- KPI types are supported
- Output columns are defined
- Session types have at least one KPI

### 2. Redis Storage (`sessionmanagement/redis_store.go`)

The configuration is stored in Redis under keys:
- `session:rules:{type}` — Configuration for each session type
- `session:config:global` — Global session settings

This allows:
- Multiple instances to access the same config
- Real-time config updates (reload without restart)
- Per-type TTL and pub/sub settings

```go
store := sessionmanagement.NewRedisSessionRulesStore(redisClient)
store.StoreSessionRulesConfig(ctx, config)
```

### 3. KPI Calculation (`sessionmanagement/kpi_calculator.go`)

The KPI calculator dynamically computes all configured KPIs based on session data:

```go
calculator := sessionmanagement.NewKPICalculator(sessionTypeConfig)
kpiResults, err := calculator.CalculateKPIs(sessionData)
// Returns: {"duration_seconds": 3600, "total_energy_kwh": 500.5, ...}
```

Supported KPI Types:
- **duration** — Time between session start and end
- **duration_filtered** — Duration where condition is true (e.g., "system_running == 1")
- **sum** — Sum of all values from source tag
- **avg** — Average of source tag values
- **max** — Peak value
- **min** — Minimum value
- **count** — Count of matching values
- **first** — First matching value
- **last** — Last matching value

### 4. Session Builder Integration

The Session Builder service uses this config to:
1. Monitor trigger tags (heat_number, shift_id, campaign_id)
2. Collect data values for all tags during the session
3. Calculate all KPIs using the dynamic calculator
4. Write computed KPIs to Parquet files
5. Publish updates to Redis pub/sub channels

## Example: Heat Session Configuration

```yaml
heat:
  trigger:
    tag_name: "heat_number"
    change_policy: "on_change"
  
  kpis:
    # Duration: from session start to end
    - name: "heat_duration_seconds"
      type: "duration"
      output_column: "duration_seconds"
    
    # Average temperature during heat
    - name: "avg_temperature"
      type: "avg"
      source_tag: "furnace_temperature"
      output_column: "avg_temperature_celsius"
    
    # Peak temperature
    - name: "max_temperature"
      type: "max"
      source_tag: "furnace_temperature"
      output_column: "max_temperature_celsius"
    
    # Total energy consumed
    - name: "total_energy"
      type: "sum"
      source_tag: "energy_kwh"
      output_column: "total_energy_kwh"
    
    # Uptime (when system is running)
    - name: "uptime_seconds"
      type: "duration_filtered"
      source_tag: "system_running"
      output_column: "uptime_seconds"
      condition: "value == 1"
  
  output_columns:
    - name: "heat_id"
      type: "string"
```

## Data Flow

1. **Raw Data Ingestion** → Tags emit values (temperature, energy, etc.)
2. **Session Trigger** → heat_number changes, starts new session
3. **Data Collection** → Values are collected for all tags during session
4. **Session End** → heat_number changes again or timeout
5. **KPI Calculation** → Dynamic calculator processes all KPIs
6. **Parquet Write** → Session data written to heat_sessions.parquet
7. **Redis Pub/Sub** → Session completion published to `heat_updates`

## Usage in Session Builder

```go
// Load config from Redis
typeConfig, err := sessionRulesStore.GetSessionTypeConfig(ctx, "heat")

// Calculate KPIs for completed session
calculator := sessionmanagement.NewKPICalculator(typeConfig)
kpis, err := calculator.CalculateKPIs(sessionData)

// Output columns are dynamically generated
for _, col := range typeConfig.OutputColumns {
    // Get value from kpis or session data
}

// Write to Parquet with dynamic schema
```

## Adding New KPI Types

To add a new KPI type (e.g., percentile):

1. Add to `SessionRulesConfig` YAML example
2. Implement calculation in `KPICalculator.calculateKPI()`
3. Add validation in `ValidateKPIType()`
4. Update dashboard to display new KPIs

Example:

```go
case "percentile_95":
    return c.calculatePercentile(kpiConfig, sessionData, 95)
```

## Configuration Hot-Reload (Future)

The system supports reloading configuration without restart:

```go
// Reload from file
newConfig, _ := sessionmanagement.LoadSessionRulesConfig("config/session-rules.yaml")

// Store updated config
sessionRulesStore.StoreSessionRulesConfig(ctx, newConfig)

// Trigger update notification
sessionRulesStore.PublishSessionUpdate(ctx, "heat", map[string]string{
    "event": "config_updated",
})
```

## Files

- `config/session-rules.yaml` — Session type definitions and KPI configurations
- `internal/sessionmanagement/config.go` — Configuration loader and validator
- `internal/sessionmanagement/redis_store.go` — Redis persistence and pub/sub
- `internal/sessionmanagement/kpi_calculator.go` — Dynamic KPI calculation engine
- `internal/sessionbuilder/` — Session tracking and Parquet writing (upcoming)

## Environment Variables

Session-related env vars (from `.env.example`):
- `SESSION_BUILDER_ENABLED=true` — Enable session builder service
- `HEAT_CHANGE_CHECK_INTERVAL_MS=1000` — How often to check for new sessions
- `SESSION_FLUSH_INTERVAL_MS=5000` — How often to flush completed sessions

## Testing the Configuration

```bash
# Validate session rules YAML
go test ./internal/sessionmanagement -v

# Load and display configuration
go run cmd/signalblocks/main.go

# Check Redis for stored config
redis-cli KEYS "session:*"
redis-cli GET "session:rules:heat"
```
