# SignalBlocks — Hot vs Cold Path Architecture

## 1. Architectural Principle

SignalBlocks is explicitly split into **Hot Path** (deterministic, ultra-low latency, no brokers) and **Cold Path** (asynchronous, resilient, integration-friendly).

> **Rule:** Nothing in the hot path may introduce GC jitter, disk contention outside controlled WAL, or broker scheduling uncertainty.

---

## 2. High-Level Diagram (Conceptual)

```
┌────────────┐     ┌─────────────────┐     ┌──────────────────┐
│  Devices   │ --> │  MQTT Broker    │ --> │ Ingest Gateway   │
│ PLC / OPC  │     │ (EMQX/Mosq)     │     │ (Go, lock-free) │
└────────────┘     └─────────────────┘     └─────────┬────────┘
                                                        │
                                                        ▼
                                              ┌──────────────────┐
                                              │ Ring Buffer /    │
                                              │ Shared Memory    │
                                              └─────────┬────────┘
                                                        │
                                                        ▼
                                              ┌──────────────────┐
                                              │ Block Builder    │
                                              │ Change-driven    │
                                              └─────────┬────────┘
                                                        │
                                   ┌────────────────────┴──────────────────┐
                                   │                                         │
                                   ▼                                         ▼
                        ┌────────────────────┐                 ┌──────────────────────┐
                        │ In-Memory Index    │                 │ Append-only WAL      │
                        │ (Query Hot Data)   │                 │ (NVMe, batched fsync)│
                        └─────────┬──────────┘                 └─────────┬────────────┘
                                   │                                         │
                                   ▼                                         ▼
                        ┌────────────────────┐                 ┌──────────────────────┐
                        │ Query Engine       │                 │ Cold Path Fanout     │
                        │ (<2ms deterministic)│                │ (Async Only)         │
                        └────────────────────┘                 └─────────┬────────────┘
                                                                            │
                                                                            ▼
                                                          ┌────────────────────────────┐
                                                          │ RabbitMQ / Kafka / NATS    │
                                                          │ Alerts / ETL / Replication │
                                                          └────────────────────────────┘
```

---

## 3. Hot Path Definition

**Purpose:** Build deterministic aggregation blocks and serve dashboards.

Included components:

* MQTT Broker (QoS 0/1, no persistence)
* Ingest Gateway
* Ring Buffer / Shared Memory
* Block Builder
* In-memory Aggregation Index
* Query Engine
* Local WAL (controlled, batched)

Explicitly excluded:

* RabbitMQ
* Kafka
* External databases
* Compression
* Network hops beyond ingress

---

## 4. Cold Path Definition

**Purpose:** Durability, integration, replay, analytics, alerts.

Components:

* Async fanout after block commit
* Message brokers (RabbitMQ / Kafka / NATS)
* Raw storage (ClickHouse, S3, MinIO)
* External consumers

Cold path failure **must not** affect hot path latency or availability.

---

# Latency Budget (Per Stage)

## 5. End-to-End Hot Path Latency Target

**Hard target:** < 2.0 ms (P99)

| Stage                  | Budget (ms) | Notes                        |
| ---------------------- | ----------- | ---------------------------- |
| MQTT receive → Gateway | 0.3         | Local network, no TLS in MVP |
| Decode + normalize     | 0.2         | Zero-copy parsing            |
| Ring buffer enqueue    | 0.05        | Lock-free                    |
| Change detection       | 0.2         | Threshold / delta checks     |
| Block construction     | 0.3         | In-memory only               |
| WAL append (batched)   | 0.2         | fsync every N ms             |
| Index update           | 0.15        | Pre-allocated structures     |
| Query availability     | 0.15        | Immediate visibility         |
| **Total**              | **1.55 ms** | Margin preserved             |

---

## 6. What Is *Not* in the Budget

* WAL replay (startup only)
* Cold path replication
* Compression
* Long-term storage writes
* Alert evaluation

These are explicitly asynchronous.

---

# Benchmark & Validation Plan

## 7. Benchmark Objectives

The benchmark suite is designed to **prove deterministic latency**, not peak throughput.

Primary goals:

* Validate **P50 / P95 / P99 latency < 2ms**
* Detect tail-latency spikes
* Verify hot-path isolation from cold-path failures
* Measure cost of WAL durability

---

## 8. Workload Profiles

### Profile A — Steady Industrial Load

* 10k points/sec
* 1k tags
* Change rate: 5–10%
* Update interval: irregular (event-driven)

### Profile B — Bursty PLC Load

* Baseline: 5k points/sec
* Bursts: 100k points/sec for 1–2 seconds
* Simulates batch PLC scans

### Profile C — Worst-case Change Storm

* 20k points/sec
* 100% change rate
* All tags crossing thresholds

---

## 9. Measurement Points

Latency timestamps captured at:

1. MQTT receive (broker)
2. Gateway ingress
3. Ring buffer enqueue
4. Block commit
5. Query-visible index update

All timestamps are monotonic (CLOCK_MONOTONIC_RAW).

---

## 10. Success Criteria

| Metric      | Target   |
| ----------- | -------- |
| P50 latency | < 1.0 ms |
| P95 latency | < 1.5 ms |
| P99 latency | < 2.0 ms |
| Max spike   | < 5 ms   |
| Data loss   | 0        |

---

## 11. Failure Injection Tests

* Kill cold-path broker during load
* Saturate disk IO with background writes
* Pause GC (stress allocation)
* Drop MQTT connections

Expected result: **Hot path latency unaffected**.

---

# Official Mermaid Diagrams

## 12. Hot vs Cold Path (Mermaid)

```mermaid
flowchart LR
    subgraph Edge
        D[Devices / PLC / OPC]
        M[MQTT Broker]
    end

    D --> M

    %% Sharded Hot Path
    M -->|hash(tag_id)| G1[Ingest Gateway #1]
    M -->|hash(tag_id)| G2[Ingest Gateway #2]
    M -->|hash(tag_id)| G3[Ingest Gateway #N]

    subgraph HotPathShard1[Hot Path Shard 1]
        G1 --> R1[Ring Buffer]
        R1 --> B1[Block Builder]
        B1 --> I1[In-Memory Index]
        B1 --> W1[Local WAL]
        I1 --> Q1[Query Engine]
    end

    subgraph HotPathShard2[Hot Path Shard 2]
        G2 --> R2[Ring Buffer]
        R2 --> B2[Block Builder]
        B2 --> I2[In-Memory Index]
        B2 --> W2[Local WAL]
        I2 --> Q2[Query Engine]
    end

    subgraph HotPathShardN[Hot Path Shard N]
        G3 --> R3[Ring Buffer]
        R3 --> B3[Block Builder]
        B3 --> I3[In-Memory Index]
        B3 --> W3[Local WAL]
        I3 --> Q3[Query Engine]
    end

    %% Cold Path
    B1 -. async .-> F[Cold Path Fanout]
    B2 -. async .-> F
    B3 -. async .-> F
    F --> MQ[RabbitMQ / Kafka / NATS]
    MQ --> E[ETL / Alerts / Replication]
```

---

# Design Doc (Investor / Accelerator Version)

## 7. Problem

Industrial dashboards depend on time-series databases that are optimized for throughput, not predictable latency. Even well-provisioned systems suffer from:

* Non-deterministic GROUP BY queries
* Cache misses on long time ranges
* Latency spikes during compaction

This makes sub-second operational decisions unreliable.

---

## 8. Solution

SignalBlocks introduces a **deterministic aggregation layer** that:

* Converts raw streams into immutable, change-driven blocks
* Eliminates runtime aggregation
* Guarantees predictable query latency (<2ms)

It augments existing databases rather than replacing them.

---

## 9. Key Technical Differentiators

* Change-driven block creation (not fixed windows)
* Pre-aggregated immutable blocks
* Memory-first architecture
* Explicit hot/cold path separation
* Brokerless hot path
* Industrial-grade semantics (state duration, integrals)

---

## 10. Why This Is Hard to Copy

* Requires deep understanding of industrial signals
* Requires latency budgeting across the entire pipeline
* Conflicts with traditional message-driven architectures
* Optimized for determinism, not just throughput

---

## 11. Target Users

* Manufacturing operations
* Energy & utilities
* Smart infrastructure
* Industrial OEM dashboards

---

## 12. Roadmap (Condensed)

* MVP: single-node deterministic engine
* Pilot: real PLC/MQTT workloads
* Scale-out: sharded hot paths + cold path replication
* Ecosystem: Grafana plugin, APIs

---

## 13. Vision

SignalBlocks becomes the **latency standard layer** for industrial analytics — enabling operators to trust what they see, instantly.

# Open Problems & Research Challenges

> This document lists **intentionally unsolved problems** in SignalBlocks.
> These are not bugs or TODOs — they are **hard system-design challenges** where expert input is valuable.
>
> If these problems look interesting to you, you are likely the kind of engineer we want to collaborate with.

---

## 1. Deterministic Query Planning over Sharded Hot Paths

**Problem**
Given a query spanning multiple tags and time ranges, determine a query plan that:

* Routes sub-queries to the minimal set of shards
* Executes shard-local queries fully in parallel
* Produces a **bounded fan-in latency** regardless of shard count

**Why this is hard**

* Multi-shard queries introduce tail-latency amplification
* Naïve fan-in behaves like scatter/gather systems (unbounded P99)
* Traditional cost-based planners optimize throughput, not determinism

**What we want to explore**

* Static fan-in limits with graceful degradation
* Time-range based shard pruning
* Deterministic merge strategies (no global sort)

---

## 2. Change-Driven Block Semantics Formalization

**Problem**
SignalBlocks creates blocks only when meaningful changes occur (thresholds, deltas, state transitions).

We need a **formal and composable model** for:

* What constitutes a "meaningful change"
* How overlapping change rules interact
* How block boundaries affect aggregation correctness

**Why this is hard**

* Industrial signals are noisy and stateful
* Change semantics differ per tag and per use case
* Incorrect modeling leads to silent analytical errors

**Open questions**

* Can change rules be expressed as a minimal algebra?
* Can we prove aggregation correctness under rule composition?

---

## 3. WAL Replay with Partial Shard Recovery

**Problem**
Each hot-path shard has a local append-only WAL.

When a shard crashes or is restarted:

* WAL replay must restore blocks and indexes
* Replay must not stall query serving longer than a bounded time

**Why this is hard**

* WALs grow large under sustained load
* Full replay is unacceptable for low-latency systems
* Partial or lazy replay risks query inconsistency

**Areas of investigation**

* Checkpoint frequency vs replay time trade-offs
* Index reconstruction without full block rehydration
* Serving queries during replay with degraded guarantees

---

## 4. Memory Pressure & Block Eviction without Latency Spikes

**Problem**
The in-memory aggregation index must evict old blocks when memory is constrained.

Requirements:

* Eviction must not cause latency spikes
* Hot blocks must remain queryable
* Evicted blocks must remain accessible via cold path

**Why this is hard**

* Traditional LRU/LFU cause lock contention
* Eviction interacts with query predictability
* Industrial workloads have skewed access patterns

**Potential directions**

* Time-segmented memory arenas
* Eviction aligned with block boundaries
* Predictive eviction based on query patterns

---

## 5. Query Determinism under Concurrent Ingestion

**Problem**
Queries must return deterministic results even while ingestion is ongoing.

This includes:

* Queries overlapping block construction
* Queries spanning block commit boundaries

**Why this is hard**

* Locking hurts latency
* Snapshotting is expensive
* Eventual consistency is unacceptable for dashboards

**Questions to solve**

* Can we define read-stability guarantees weaker than full snapshot isolation but still deterministic?
* Can block immutability be leveraged for lock-free reads?

---

## 6. Hot–Cold Path Consistency Guarantees

**Problem**
Hot path serves dashboards; cold path handles replication, storage, and analytics.

We must define:

* What consistency guarantees exist between hot and cold paths
* How divergence is detected and reconciled

**Why this is hard**

* Cold path is asynchronous by design
* Replays and retries introduce reordering

**Non-goals (explicit)**

* Strong consistency across paths
* Exactly-once semantics end-to-end

---

## 7. Proving Latency Bounds (Not Just Measuring Them)

**Problem**
Most systems benchmark latency; SignalBlocks aims to **reason about it**.

We want:

* A clear argument for why P99 latency is bounded
* Identification of all unbounded operations

**Why this is hard**

* OS scheduling
* GC behavior
* Hidden allocations

**Research directions**

* Static latency budgets per component
* Allocation-free critical paths
* Formal identification of jitter sources

---

## 8. Shard Rebalancing without Data Movement (Future)

**Problem**
As workloads change, shard load becomes imbalanced.

Goal:

* Rebalance ingestion and query load
* Without moving historical blocks

**Why this is hard**

* Tag-to-shard mapping is fundamental
* Moving blocks breaks determinism guarantees

**Early ideas**

* Virtual shards
* Dual-write during transition windows

---

## How to Contribute

We are especially interested in collaborators who:

* Enjoy systems-level reasoning
* Care about tail latency, not averages
* Prefer explicit trade-offs over magic abstractions

If any of these problems resonate with you, open an issue or start a discussion.
# 3.WAL Replay with Partial Shard Recovery

> This document deep-dives into **Open Problem #3** from `open-problems.md`.
> It describes constraints, failure modes, non-goals, and candidate designs.
>
> Status: **Exploratory / Design-in-progress**

---

## 1. Problem Restatement

Each SignalBlocks hot-path shard maintains a **local append-only WAL** containing:

* Block creation records
* Block finalization markers
* Index update hints (optional)

When a shard crashes or restarts, the system must:

1. Restore queryable state from WAL
2. Resume ingestion
3. Preserve deterministic latency guarantees

**Key requirement:** WAL replay must not introduce unbounded startup time or latency spikes.

---

## 2. Constraints (Hard Requirements)

### 2.1 Determinism First

* Replay behavior must be predictable
* Replay must not block unrelated shards
* No global coordination during replay

### 2.2 Bounded Recovery Time

* Full WAL replay on restart is unacceptable
* Recovery time must scale with *recent* activity, not total history

### 2.3 Query Availability During Replay

* Queries must remain available
* Temporary degradation is acceptable
* Silent inconsistency is not

---

## 3. Failure Scenarios

### Scenario A — Clean Restart

* Process shutdown
* WAL closed correctly

**Expectation:** Fast replay, minimal degradation

---

### Scenario B — Crash During Block Construction

* Block partially written
* No finalization marker

**Expectation:**

* Partial block discarded
* No query visibility

---

### Scenario C — Crash After Block Commit, Before Index Update

* Block durable in WAL
* Index missing block reference

**Expectation:**

* Block must become query-visible after replay

---

### Scenario D — Large WAL Accumulation

* High sustained load
* Long uptime

**Expectation:**

* Replay time remains bounded

---

## 4. Non-Goals (Explicit)

To avoid accidental complexity, SignalBlocks does **not** aim to provide:

* Exactly-once semantics across shards
* Synchronous replication during replay
* Historical re-materialization during startup

---

## 5. Design Direction 1 — Checkpointed Replay

### Concept

Periodically emit **checkpoint records** into the WAL:

* Fully materialized index snapshot
* Last committed block ID

On restart:

* Load latest checkpoint
* Replay WAL entries after checkpoint only

### Trade-offs

**Pros:**

* Bounded replay time
* Simple mental model

**Cons:**

* Checkpoint creation cost
* Memory pressure during snapshot

---

## 6. Design Direction 2 — Two-Tier WAL (Hot + Cold)

### Concept

Split WAL into:

* Hot WAL: recent blocks
* Cold WAL: archived, immutable segments

On restart:

* Replay hot WAL only
* Cold WAL accessed lazily if needed

### Trade-offs

**Pros:**

* Replay cost proportional to recent activity
* Natural fit with hot/cold philosophy

**Cons:**

* Segment management complexity
* Requires careful cutoff semantics

---

## 7. Design Direction 3 — Index-First Recovery

### Concept

Persist **index snapshots** separately from raw blocks.

On restart:

* Load index snapshot
* Validate against WAL
* Skip block-level replay unless inconsistency detected

### Risks

* Snapshot corruption
* Index/WAL divergence

---

## 8. Serving Queries During Replay

### Degraded Modes

During replay, shards may enter one of the following modes:

* **READ_ONLY_STABLE**: Queries served from last known-good index
* **READ_WITH_GAPS**: Missing recent blocks, explicit warnings
* **WARMING_UP**: Queries delayed but bounded

The mode must be visible to the query planner.

---

## 9. Replay Correctness Rules

The following invariants must hold:

1. No partially constructed block is ever visible
2. Block order is preserved per shard
3. Replayed blocks produce identical aggregation results

Violations are considered fatal.

---

## 10. Open Research Questions

* What is the minimal metadata needed to resume queries safely?
* Can replay be interleaved with live ingestion?
* Is there a provable upper bound on replay time under worst-case load?

---

## 11. Why This Matters

WAL replay is where many low-latency systems silently fail:

* Long warm-up times
* Latency spikes after restart
* Inconsistent dashboards

Solving this cleanly is critical to SignalBlocks’ credibility as a **deterministic system**.

---

## 12. Invitation

If you have experience with:

* Log-structured systems
* Recovery protocols
* Low-latency indexing

We welcome discussion and alternative designs.
# 3.WAL Replay with Partial Shard Recovery

> This document deep-dives into **Open Problem #3** from `open-problems.md`.
> It describes constraints, failure modes, non-goals, and candidate designs.
>
> Status: **Exploratory / Design-in-progress**

---

## 1. Problem Restatement

Each SignalBlocks hot-path shard maintains a **local append-only WAL** containing:

* Block creation records
* Block finalization markers
* Index update hints (optional)

When a shard crashes or restarts, the system must:

1. Restore queryable state from WAL
2. Resume ingestion
3. Preserve deterministic latency guarantees

**Key requirement:** WAL replay must not introduce unbounded startup time or latency spikes.

---

## 2. Constraints (Hard Requirements)

### 2.1 Determinism First

* Replay behavior must be predictable
* Replay must not block unrelated shards
* No global coordination during replay

### 2.2 Bounded Recovery Time

* Full WAL replay on restart is unacceptable
* Recovery time must scale with *recent* activity, not total history

### 2.3 Query Availability During Replay

* Queries must remain available
* Temporary degradation is acceptable
* Silent inconsistency is not

---

## 3. Failure Scenarios

### Scenario A — Clean Restart

* Process shutdown
* WAL closed correctly

**Expectation:** Fast replay, minimal degradation

---

### Scenario B — Crash During Block Construction

* Block partially written
* No finalization marker

**Expectation:**

* Partial block discarded
* No query visibility

---

### Scenario C — Crash After Block Commit, Before Index Update

* Block durable in WAL
* Index missing block reference

**Expectation:**

* Block must become query-visible after replay

---

### Scenario D — Large WAL Accumulation

* High sustained load
* Long uptime

**Expectation:**

* Replay time remains bounded

---

## 4. Non-Goals (Explicit)

To avoid accidental complexity, SignalBlocks does **not** aim to provide:

* Exactly-once semantics across shards
* Synchronous replication during replay
* Historical re-materialization during startup

---

## 5. Design Direction 1 — Checkpointed Replay

### Concept

Periodically emit **checkpoint records** into the WAL:

* Fully materialized index snapshot
* Last committed block ID

On restart:

* Load latest checkpoint
* Replay WAL entries after checkpoint only

### Trade-offs

**Pros:**

* Bounded replay time
* Simple mental model

**Cons:**

* Checkpoint creation cost
* Memory pressure during snapshot

---

## 6. Design Direction 2 — Two-Tier WAL (Hot + Cold)

### Concept

Split WAL into:

* Hot WAL: recent blocks
* Cold WAL: archived, immutable segments

On restart:

* Replay hot WAL only
* Cold WAL accessed lazily if needed

### Trade-offs

**Pros:**

* Replay cost proportional to recent activity
* Natural fit with hot/cold philosophy

**Cons:**

* Segment management complexity
* Requires careful cutoff semantics

---

## 7. Design Direction 3 — Index-First Recovery

### Concept

Persist **index snapshots** separately from raw blocks.

On restart:

* Load index snapshot
* Validate against WAL
* Skip block-level replay unless inconsistency detected

### Risks

* Snapshot corruption
* Index/WAL divergence

---

## 8. Serving Queries During Replay

### Degraded Modes

During replay, shards may enter one of the following modes:

* **READ_ONLY_STABLE**: Queries served from last known-good index
* **READ_WITH_GAPS**: Missing recent blocks, explicit warnings
* **WARMING_UP**: Queries delayed but bounded

The mode must be visible to the query planner.

---

## 9. Replay Correctness Rules

The following invariants must hold:

1. No partially constructed block is ever visible
2. Block order is preserved per shard
3. Replayed blocks produce identical aggregation results

Violations are considered fatal.

---

## 10. Open Research Questions

* What is the minimal metadata needed to resume queries safely?
* Can replay be interleaved with live ingestion?
* Is there a provable upper bound on replay time under worst-case load?

---

## 11. Why This Matters

WAL replay is where many low-latency systems silently fail:

* Long warm-up times
* Latency spikes after restart
* Inconsistent dashboards

Solving this cleanly is critical to SignalBlocks’ credibility as a **deterministic system**.

---

## 12. Invitation

If you have experience with:

* Log-structured systems
* Recovery protocols
* Low-latency indexing

We welcome discussion and alternative designs.


for continuity: (raj)

Pseudo-protocol برای WAL records

Timeline diagram از crash → replay → serve

Why WAL replay breaks determinism in most systems (doc تبلیغاتی-فنی)