# SignalBlocks

**Deterministic ultra-low latency analytics for industrial time-series data**

SignalBlocks is a real-time analytics accelerator that sits on top of existing time-series databases (ClickHouse, InfluxDB, etc.) to deliver predictable **<2ms** query latency for repetitive industrial dashboards — no more non-deterministic GROUP BY or raw data scans.

Built for manufacturing, energy, utilities, and Industry 4.0/5.0 environments handling high-frequency sensor/PLC/OPC/MQTT data.

[![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Status: Pre-Alpha](https://img.shields.io/badge/Status-Pre--Alpha-red)](https://github.com/masoudrajaei/signalblocks)

## ✨ Core Features (MVP in progress)

- **Change-driven immutable blocks** — blocks created only on meaningful data changes (delta/threshold-based), not fixed intervals
- **In-memory aggregation index** — deterministic latency for common KPIs (time-weighted avg, duration per state, min/max, integrals)
- **Metadata-driven tag model** — per-tag aggregation rules, grouping, and change policies
- **No runtime GROUP BY** — pre-aggregated blocks eliminate expensive queries on large time ranges
- **Hot/Warm/Cold storage paths** — RAM → NVMe (light compression) → S3/MinIO (aggressive compression)
- Integrates with Grafana, Power BI, custom UIs

## Why SignalBlocks?

Existing time-series DBs (ClickHouse, Influx, Timescale) excel at ingestion & throughput, but struggle with **predictable low-latency** for repetitive dashboard workloads in industrial settings.

SignalBlocks acts as an **accelerator layer** — keep your raw data for audits/deep analysis, but serve 99% of operational dashboards from fast, immutable aggregated blocks.

## Proposed Architecture (Recommended for Real-time & Low-Latency)

To achieve **sub-2ms deterministic latency** for dashboards while handling high-frequency industrial data ingestion, the following dual-path architecture is recommended:

- **WebSocket / Streaming** for continuous, low-latency ingest & live push to dashboards
- **REST / gRPC API** for on-demand queries, configuration, and integration with tools

```mermaid
graph TD
    %% ======================
    %% Input / Edge
    %% ======================
    MQTT[MQTT Broker] --> IG1[Ingest Gateway 1]
    MQTT --> IG2[Ingest Gateway 2]
    MQTT --> IGN[Ingest Gateway N]

    REST[REST/gRPC/WS] --> IG1
    REST --> IG2
    REST --> IGN

    %% ======================
    %% Hot Path Shard 1
    %% ======================
    IG1 --> RB1[Ring Buffer Lock-free]
    RB1 --> BB1[Block Builder Change-driven]
    BB1 --> WAL1[Append-only WAL NVMe]
    BB1 --> IDX1[In-Memory Index]
    IDX1 --> Q1[Query Engine <2ms]

    %% ======================
    %% Hot Path Shard 2
    %% ======================
    IG2 --> RB2[Ring Buffer Lock-free]
    RB2 --> BB2[Block Builder Change-driven]
    BB2 --> WAL2[Append-only WAL NVMe]
    BB2 --> IDX2[In-Memory Index]
    IDX2 --> Q2[Query Engine <2ms]

    %% ======================
    %% Hot Path Shard N
    %% ======================
    IGN --> RBN[Ring Buffer Lock-free]
    RBN --> BBN[Block Builder Change-driven]
    BBN --> WALN[Append-only WAL NVMe]
    BBN --> IDXN[In-Memory Index]
    IDXN --> QN[Query Engine <2ms]

    %% ======================
    %% Cold Path
    %% ======================
    BB1 -.-> FANOUT[Cold Path Fanout]
    BB2 -.-> FANOUT
    BBN -.-> FANOUT

    FANOUT --> MQ[Kafka / RabbitMQ / NATS]
    MQ --> EXT[ETL / Alerts / Long-term Storage]

    %% ======================
    %% Styling with text color
    %% ======================
    style IG1 fill:#a3c1e1,stroke:#333,color:#000,stroke-width:1.5px
    style IG2 fill:#a3c1e1,stroke:#333,color:#000,stroke-width:1.5px
    style IGN fill:#a3c1e1,stroke:#333,color:#000,stroke-width:1.5px

    style RB1 fill:#c4d6eb,stroke:#333,color:#000
    style RB2 fill:#c4d6eb,stroke:#333,color:#000
    style RBN fill:#c4d6eb,stroke:#333,color:#000

    style BB1 fill:#d0e0f0,stroke:#333,color:#000
    style BB2 fill:#d0e0f0,stroke:#333,color:#000
    style BBN fill:#d0e0f0,stroke:#333,color:#000

    style WAL1 fill:#e0ebf5,stroke:#333,color:#000
    style WAL2 fill:#e0ebf5,stroke:#333,color:#000
    style WALN fill:#e0ebf5,stroke:#333,color:#000

    style IDX1 fill:#d0e0f0,stroke:#333,color:#000
    style IDX2 fill:#d0e0f0,stroke:#333,color:#000
    style IDXN fill:#d0e0f0,stroke:#333,color:#000

    style Q1 fill:#c4d6eb,stroke:#333,color:#000
    style Q2 fill:#c4d6eb,stroke:#333,color:#000
    style QN fill:#c4d6eb,stroke:#333,color:#000

    style FANOUT fill:#b0b8c0,stroke:#666,stroke-dasharray:5 5,color:#000
    style MQ fill:#b0b8c0,stroke:#666,color:#000
    style EXT fill:#b0b8c0,stroke:#666,color:#000

```

Key Benefits of this Architecture:

WebSocket/MQTT → minimal latency for continuous high-frequency ingestion from industrial devices
In-memory index → serves both live updates (push) and fast queries
Dual output → live real-time dashboards via WebSocket + flexible on-demand access via REST/gRPC
Scalable & efficient — Go's concurrency model (goroutines) handles thousands of connections with low overhead

Implementation Notes (Go):

Use github.com/gorilla/websocket for WebSocket endpoints
Use github.com/gin-gonic/gin or net/http for REST
Optional: github.com/grpc/grpc-go for gRPC if type-safety & streaming is needed
Global broadcaster (channel) to push new aggregations to all connected dashboard clients

This architecture ensures SignalBlocks remains ultra-low latency while being flexible for different use cases.

Main data flow:

Raw data is ingested and stored in Raw Storage for audit and deep analysis
Change-driven blocks are created only on meaningful changes (not fixed time intervals)
Immutable aggregated blocks feed the in-memory index
Query planner serves sub-2ms responses directly from the index to dashboards

Current Status

MVP in progress (built with Go)
Ingestion API (OPC/MQTT simulation)
Change-driven Block Builder
In-memory Aggregation Index (Redis-like)
Basic query planner & Grafana integration demo

Pre-seed stage — actively developed by founder with years of hands-on industrial BI/time-series experience
Targeting first pilot use cases in 2026

Quick Start (once MVP ready)
Bash# Coming soon — example when prototype is runnable
git clone https://github.com/masoudrajaei/signalblocks.git
cd signalblocks
go run cmd/signalblocks/main.go --config=config.yaml

(Currently the repo is at skeleton stage — follow updates!)
Roadmap (2026)

Q1: MVP prototype + sample industrial dataset demo
Q2: Pilot validation with real use cases
Q3: Open API + Grafana datasource plugin
Q4: Early community contributions + benchmarks vs. raw DB queries

Contributing
Contributions welcome! This is early stage — issues, PRs, ideas for industrial use cases are all appreciated.
See CONTRIBUTING.md (coming soon).
License
MIT License — see LICENSE

Contact / Founder
Masoud Rajaei
Founder @ MetaBlock Systems
mr.rajjaei@gmail.com
+989135605562
Open to collaboration, pilots, accelerators in world
Built with ❤️ for smarter industrial decisions.

