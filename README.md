# TradeRelay — Real-Time Trading Execution & Copy-Trading Platform

TradeRelay is a Go-based backend system for exploring the infrastructure behind real-time trade execution, broker integrations, and multi-account copy-trading workflows.

The project is centered on a practical systems problem:

> **How can a trading platform coordinate execution across multiple accounts while maintaining correctness under concurrency, retries, partial fills, timeouts, and failures?**


## Architecture

```text
Strategy Signal
      |
      v
Signal Validation
      |
      v
Account & Risk Checks
      |
      v
Copy-Trade Allocation
      |
      v
Execution Coordinator
      |
      v
Broker Adapter Layer
      |
      v
Orders / Fills / Rejections
      |
      v
Position Reconciliation
      |
      +--------------------+
      |                    |
      v                    v
   REST API         WebSocket Updates
```

## Why This Is a Systems Problem

Executing one order is relatively simple. Coordinating execution across many accounts becomes more difficult once concurrency and external systems are involved.

A single strategy signal may need to produce orders for many follower accounts, each with different balances, allocation percentages, positions, and risk limits.

Execution can also fail in ambiguous ways. A broker request may time out even though the broker accepted the order. One account may receive a full fill while another receives a partial fill or rejection. Execution events may arrive late or out of order. The service may restart while orders remain unresolved.

TradeRelay is designed around handling these conditions safely rather than assuming the execution path always succeeds.

## Core Components

### Signal Service

Receives strategy signals, validates required fields, and rejects malformed or duplicate signals before they enter the execution pipeline.

### Account & Risk Layer

Maintains follower account state and evaluates whether an account is eligible to participate in an execution based on configurable allocation and exposure constraints.

### Allocation Engine

Transforms a single strategy signal into account-specific execution intents.

For example:

```text
Strategy Signal: BUY AAPL

Follower A → 100 shares
Follower B → 40 shares
Follower C → not eligible
```

Allocation decisions can depend on account configuration, available capital, allocation percentage, and risk limits.

### Execution Coordinator

Coordinates account-specific orders and tracks their lifecycle through submission, acknowledgement, execution, rejection, and recovery.

The coordinator is responsible for preventing duplicate broker actions and maintaining consistent execution state.

### Broker Adapter Layer

Provides a stable internal interface around broker-specific APIs.

```text
Execution Coordinator
        |
        v
    Broker Interface
      /     |      \
     v      v       v
 Broker A Broker B Simulator
```

Initial development uses deterministic broker simulators so latency, rejection, timeout, and partial-fill scenarios can be reproduced safely.

### Reconciliation Engine

Compares TradeRelay's internal order and position state against broker-reported state.

Reconciliation is particularly important when a request produces an ambiguous result—for example, when the broker accepts an order but the network response is lost.

### API & Real-Time Layer

REST endpoints expose platform state and execution operations, while WebSocket connections provide real-time order, fill, and position updates to operational clients.

## Execution Lifecycle

```text
Signal Received
      |
      v
Validated
      |
      v
Accounts Selected
      |
      v
Allocation Calculated
      |
      v
Orders Created
      |
      v
Broker Submission
      |
      +------> Rejected
      |
      +------> Partial Fill
      |
      +------> Filled
      |
      +------> Unknown / Timeout
                    |
                    v
               Reconciliation
```

The `Unknown / Timeout` state is important. A timeout does not necessarily mean an order failed. Blindly retrying an ambiguous request could result in duplicate execution.

TradeRelay therefore treats **idempotency, durable state, and reconciliation** as core execution requirements.

## Reliability Model

The system is designed to test failure scenarios including:

* duplicate signal delivery
* concurrent duplicate requests
* broker timeouts
* partial fills
* rejected orders
* delayed broker responses
* out-of-order execution events
* service restart during active execution
* database failures
* slow downstream consumers
* burst traffic
* broker/internal state disagreement

Correctness takes priority over raw throughput. Performance optimization is introduced only after execution state and recovery behavior are deterministic and tested.

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
   +------> Allocation ------> Account
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

Each layer represents a different responsibility.

A signal represents strategy intent. An execution represents TradeRelay's attempt to realize that intent. Allocations determine account participation. Orders represent broker-facing actions. Fills represent actual execution results, and positions represent the resulting account state.

## Concurrency & Idempotency

Trading infrastructure receives concurrent requests and cannot assume messages arrive exactly once.

TradeRelay therefore uses synchronization and idempotency controls around shared state and external execution boundaries.

For example, if two requests containing the same signal arrive simultaneously:

```text
Request A ----\
               ---> TradeRelay ---> one execution
Request B ----/
```

the system should create only one logical execution.

Concurrency tests and Go's race detector are used to verify that shared-state behavior remains safe under simultaneous access.

## Technology

**Backend**

* Go
* Go standard library
* REST APIs
* WebSockets

**Data**

* MySQL
* transactional persistence
* schema migrations

**Infrastructure**

* Docker
* health endpoints
* structured logging
* metrics

**Testing**

* Go unit tests
* concurrency tests
* race detection
* integration tests
* failure injection
* load testing

Additional infrastructure will be introduced only when it supports a concrete systems requirement.

## Evaluation

TradeRelay will be evaluated using measured behavior rather than theoretical performance claims.

Planned measurements include:

* request throughput
* execution throughput
* p50 / p95 / p99 latency
* allocation latency as follower count increases
* concurrent duplicate-request behavior
* broker timeout recovery
* reconciliation correctness
* burst-load behavior
* database contention
* recovery after process failure

Performance results will be documented alongside the workload and environment used to produce them.

## Current Status

**Early development.**

Implemented:

* Go project structure
* trading signal domain model
* BUY/SELL signal types
* signal validation
* validation unit tests
* concurrency-safe in-memory signal storage
* duplicate signal detection

Current development is focused on concurrency testing and idempotent signal ingestion before introducing follower accounts and allocation.

## Development Roadmap

```text
Signal Model & Validation
          |
          v
Idempotent Signal Storage
          |
          v
Account Model
          |
          v
Copy-Trade Allocation
          |
          v
Execution State Machine
          |
          v
Broker Adapter
          |
          v
Failure & Retry Handling
          |
          v
MySQL Persistence
          |
          v
Reconciliation
          |
          v
REST / WebSocket APIs
          |
          v
Observability
          |
          v
Load & Failure Evaluation
```



