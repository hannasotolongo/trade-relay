cat > README.md <<'EOF'
# Pulse — Real-Time Trading Execution & Copy-Trading Platform

Pulse is a Go-based backend systems project exploring the engineering challenges behind real-time trading infrastructure, broker integrations, copy-trading workflows, and reliable execution.

The project is built around a practical systems question:

> **How can a trading platform coordinate execution across many accounts while preserving correctness under retries, partial fills, failures, and changing account state?**

## Architecture

```text
Strategy Signal
      |
      v
Signal Validation
      |
      v
Risk & Account Checks
      |
      v
Copy-Trade Allocation
      |
      v
Execution Coordinator
      |
      v
Broker Adapter
      |
      v
Orders / Fills / Rejections
      |
      v
Position Reconciliation
      |
      +------------------+
      |                  |
      v                  v
   REST API       WebSocket Updates
```

## Engineering Focus

Pulse is designed around backend problems that occur in financial systems:

- concurrent execution across multiple accounts
- proportional copy-trade allocation
- broker API abstraction
- order lifecycle management
- partial fills and rejected orders
- idempotent request handling
- retries and timeout recovery
- transactional state management
- position and execution reconciliation
- real-time event delivery
- auditability and event history
- backpressure and overload behavior
- observability and health monitoring
- latency and throughput measurement

## Execution Model

A strategy produces a trading signal:

```text
BUY AAPL
```

Pulse validates the signal and determines which follower accounts are eligible to participate.

Each account may have a different balance, allocation percentage, exposure, or risk limit. The allocation engine calculates account-specific orders before the execution coordinator submits them through a broker adapter.

The system is designed around the fact that execution is imperfect.

A broker request may time out after the broker already accepted the order. One account may receive a partial fill while another is rejected. Execution events may arrive asynchronously or out of order. A service may restart while orders remain unresolved.

Pulse treats these conditions as normal distributed-system behavior rather than exceptional edge cases.

## Planned Components

### Signal Service

Receives and validates incoming strategy signals while preventing duplicate signal processing.

### Account & Risk Layer

Maintains follower account state and applies eligibility, allocation, and exposure constraints before execution.

### Allocation Engine

Transforms one strategy signal into account-specific execution intents according to follower configuration and available account state.

### Execution Coordinator

Coordinates order submission, tracks execution state, handles timeouts, and prevents duplicate broker actions.

### Broker Adapter Layer

Provides a stable internal interface around broker-specific APIs. Simulated brokers will initially allow latency, rejection, timeout, and partial-fill behavior to be tested deterministically.

### Reconciliation Engine

Compares internal order and position state against broker-reported state and identifies inconsistencies requiring recovery.

### API & Real-Time Layer

Exposes signals, accounts, executions, orders, positions, and system health through REST APIs and WebSocket updates.

## Data Model

```text
Strategy
   |
   v
Signal
   |
   v
Execution
   |
   +----> Allocation ----> Account
   |
   v
Order
   |
   v
Fill
   |
   v
Position
```

## Correctness & Failure Scenarios

Pulse will explicitly test:

- duplicate signal delivery
- concurrent duplicate requests
- broker timeouts
- partial fills
- rejected orders
- delayed execution events
- out-of-order events
- process failure during execution
- database failures
- slow downstream consumers
- burst traffic
- disagreement between broker and internal state

A central design principle is that **execution correctness takes priority over raw throughput**. Optimization will be introduced only after baseline state transitions and recovery behavior are measurable and tested.

## Technology

- **Go** — primary backend implementation
- **MySQL** — persistent transactional state
- **REST APIs** — platform integrations
- **WebSockets** — real-time execution updates
- **Docker** — reproducible deployment
- **Go concurrency primitives** — concurrent execution workflows
- **Structured logging and metrics** — operational visibility

Infrastructure and frontend components will be added only where they support the backend system.

## Evaluation

Performance and reliability claims will be based on measured behavior.

Planned evaluation includes:

- request throughput
- execution throughput
- p50 / p95 / p99 latency
- allocation latency as follower count increases
- duplicate-request behavior
- broker failure recovery
- burst-load behavior
- reconciliation accuracy
- concurrent execution correctness

Go's race detector and concurrency tests will also be used to identify unsafe shared-state behavior.

## Current Status

**Early development.**

Currently implemented:

- Go project structure
- trading signal domain model
- BUY/SELL type constraints
- signal validation
- validation unit tests
- concurrency-safe in-memory signal storage
- duplicate signal detection

Current development is focused on testing concurrent signal ingestion before introducing account allocation and execution state.

## Development Principles

1. Correctness before optimization.
2. Explicit state transitions.
3. Idempotency at external boundaries.
4. Broker APIs are unreliable boundaries.
5. Persistent state remains authoritative.
6. Failures should be reproducible in tests.
7. Performance claims require measurement.

