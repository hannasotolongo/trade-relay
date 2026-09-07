# TradeRelay

**Reliable Trade Execution and Multi-Account Order Coordination in Go**

TradeRelay is a backend trading system designed to coordinate reliable order execution across multiple brokerage accounts.

The system receives trading signals, validates and allocates them across follower accounts, creates durable orders, and coordinates execution through a broker abstraction. Its design focuses on the failure cases that make trading infrastructure difficult: duplicate submissions, ambiguous broker responses, application crashes, concurrent state updates, partial execution, and divergence between internal and broker state.

TradeRelay uses persistent MySQL order state, stable client order IDs, explicit execution state transitions, optimistic concurrency control, reconciliation, and startup recovery to maintain consistent order state across failures.

> **Current scope:** TradeRelay uses a simulated broker and local MySQL environment to test execution and failure-recovery behavior. It does not currently connect to a live brokerage or execute real trades.


## Results

The current implementation has been validated through unit, concurrency, persistence, and real-MySQL integration tests.

| Validation | Result |
|---|---|
| Concurrent duplicate submission | 20 concurrent submissions using the same client order ID resolved to a single simulated broker order |
| Stale-write protection | Version-based optimistic concurrency rejected an update after another writer advanced the persisted order version |
| Interrupted submission recovery | A persisted `SUBMITTING` order was recovered after a simulated application restart by querying broker state using its stable client order ID |
| Startup recovery | Persisted `SUBMITTING` and `UNKNOWN` orders were automatically discovered and reconciled against broker state |
| Terminal-state protection | `FILLED` orders remained unchanged during startup recovery |
| Persistent execution lifecycle | Order creation, submission, broker acknowledgement, version advancement, and recovery were validated against MySQL 8.4 |
| Concurrent in-memory execution | Go race-detector tests pass across the current test suite |

These results validate **correctness and failure-recovery behavior**, not production trading performance. Throughput and latency benchmarking have not yet been performed.

## The Problem

Submitting an order to a broker is straightforward when every component behaves correctly. The difficult part is maintaining a consistent view of that order when the application, network, database, or broker fails at different points in the execution lifecycle.

A trade submission crosses two independent systems: TradeRelay's internal state and the broker's execution system. These systems cannot be updated atomically. Once a broker accepts an order, a local database transaction cannot undo that external action.

This creates an important failure window.

Consider an order to buy 100 shares of AAPL:

```text
TradeRelay                              Broker
    |                                     |
    |  Persist order as SUBMITTING        |
    |                                     |
    |---- BUY 100 AAPL ------------------>|
    |                                     |
    |                              Order accepted
    |                              Broker ID: 789
    |                                     |
    |<---- ACKNOWLEDGED / broker-789 -----|
    |
    X  Application crashes before
       acknowledgement is saved to MySQL
```

The broker has successfully accepted the order, but TradeRelay never persisted the acknowledgement.

After the application restarts, the two systems disagree:

```text
TradeRelay / MySQL                   Broker

Order: order-123                     Order: broker-789
Status: SUBMITTING                   Status: ACKNOWLEDGED
BrokerOrderID: unknown               ClientOrderID: order-123
```

TradeRelay now has a dangerous decision to make.

If it assumes the submission failed and sends the order again, the broker could receive a duplicate:

```text
Original submission    BUY 100 AAPL
Blind retry            BUY 100 AAPL
                       ------------
Potential exposure     BUY 200 AAPL
```

If TradeRelay instead assumes the order succeeded without verifying it, its internal state may remain inconsistent with the broker.

A timeout creates the same problem. A timeout only means TradeRelay did not receive a definitive response; it does not prove that the broker rejected or never received the order.

TradeRelay therefore treats execution as a state-coordination problem rather than a single broker API call.

The system is designed around several guarantees:

- **Durable state:** execution state is persisted in MySQL so it survives application restarts.
- **Stable order identity:** each logical order has a stable client order ID that can be used to identify the same order across retries and recovery.
- **Explicit uncertainty:** ambiguous submission outcomes are represented as `UNKNOWN` rather than incorrectly treated as failures.
- **Idempotent submission behavior:** repeated submissions of the same client order ID must not create independent logical orders.
- **Reconciliation:** TradeRelay can query the broker to determine the authoritative external state of an order.
- **Crash recovery:** unresolved orders can be discovered from persistent storage when the application restarts.
- **Concurrency protection:** version-based optimistic concurrency prevents stale processes from silently overwriting newer order state.

Together, these mechanisms allow TradeRelay to recover from failures without assuming that every network request receives a clean response or that only one process can modify an order.

---

## Architecture

TradeRelay separates trading intent, account allocation, durable order state, broker execution, and failure recovery into distinct stages.

```text
                         TradeRelay

 Strategy Signal
       |
       v
 +----------------------+
 | Signal Validation    |
 +----------------------+
       |
       v
 +----------------------+
 | Multi-Account        |
 | Allocation           |
 +----------------------+
       |
       v
 +----------------------+
 | Order Creation       |
 +----------------------+
       |
       v
 +----------------------+
 | MySQL                |
 | Durable Order State  |
 +----------------------+
       |
       v
 +----------------------+
 | Execution            |
 | Coordinator          |
 +----------------------+
       |
       v
 +----------------------+
 | Broker Adapter       |
 +----------------------+
       |
       v
 +----------------------+
 | External Broker      |
 +----------------------+
       |
       | acknowledgements
       | fills
       | rejections
       | order lookups
       v
 +----------------------+
 | Reconciliation       |
 | & Recovery           |
 +----------------------+
       |
       v
 +----------------------+
 | MySQL                |
 | Updated Order State  |
 +----------------------+
```

### Execution Flow

A strategy produces a signal describing the intended trade:

```text
BUY 100 AAPL
```

TradeRelay validates the signal and calculates account-specific allocations based on each follower account's configuration.

```text
                   BUY 100 AAPL
                        |
            +-----------+-----------+
            |           |           |
            v           v           v
        Account A   Account B   Account C
        BUY 50      BUY 30      BUY 20
```

Each allocation becomes an independent order with its own account identity, quantity, execution state, version, and stable client order ID.

Before external execution begins, the order is persisted in MySQL. This creates a durable record that survives application failure.

The execution coordinator then moves the order into `SUBMITTING` and persists that transition before calling the broker.

```text
CREATED
   |
   | persist
   v
SUBMITTING
   |
   | broker submission
   v
External Broker
```

Broker communication is isolated behind an interface rather than being embedded directly into the execution logic:

```go
type Broker interface {
    SubmitOrder(
        ctx context.Context,
        order Order,
    ) (BrokerResult, error)

    GetOrder(
        ctx context.Context,
        brokerOrderID string,
    ) (BrokerResult, error)

    GetOrderByClientID(
        ctx context.Context,
        clientOrderID string,
    ) (BrokerResult, error)
}
```

`SubmitOrder` handles execution, while the lookup operations allow TradeRelay to determine broker state later if the original submission response is lost or the application crashes.

The current implementation uses a stateful simulated broker. It preserves broker-side order identity and supports deterministic testing of duplicate submissions, failures, ambiguous outcomes, reconciliation, and recovery without executing real trades.

### Durable State Boundary

MySQL acts as the durable state boundary between transient application execution and persistent order history.

```text
         TradeRelay Process

        application memory
               |
               | lost on crash
               v
    -------------------------
             MySQL
    -------------------------
               |
               | survives restart
               v
       durable order state
```

This boundary is important because broker execution is an external side effect.

TradeRelay can roll back a failed database transaction, but it cannot roll back an order that an external broker has already accepted simply by reverting local database state.

The system therefore persists execution progress and maintains enough order identity to recover after failure. When local state is uncertain, TradeRelay queries the broker and reconciles the persisted order against the broker's authoritative execution state.

## Order State Machine

TradeRelay models execution as an explicit state machine so order state cannot change arbitrarily.

```text
CREATED
   |
   v
SUBMITTING
   |------------------|
   v                  v
ACKNOWLEDGED       UNKNOWN
   |                  |
   v                  |----> ACKNOWLEDGED
PARTIALLY_FILLED      |----> PARTIALLY_FILLED
   |                  |----> FILLED
   v                  |----> REJECTED
FILLED
```

`REJECTED` and `FILLED` are terminal states.

`UNKNOWN` represents an ambiguous broker outcome, such as a timeout where TradeRelay cannot determine whether the broker received the order. Rather than retrying blindly, the order is reconciled against broker state.

Invalid transitions are rejected by the state machine, preventing inconsistent lifecycle changes such as moving a completed order back into execution.

---

## Reliability & Failure Recovery

TradeRelay is designed around the assumption that broker requests, database operations, and application processes can fail independently.

### Idempotent Execution

Each order has a stable client order ID. Repeated submissions using the same ID resolve to the same logical broker order, reducing the risk of duplicate execution during retries.

### Reconciliation

When execution state is uncertain, TradeRelay queries the broker using either the broker order ID or stable client order ID and updates local state from the broker's authoritative result.

### Crash Recovery

On restart, TradeRelay scans MySQL for unresolved `SUBMITTING` and `UNKNOWN` orders and reconciles them before treating their outcomes as final.

```text
Application Restart
        |
        v
Load SUBMITTING / UNKNOWN
        |
        v
Query Broker
        |
        v
Reconcile State
        |
        v
Persist to MySQL
```

This allows an order accepted immediately before an application crash to be recovered without blindly resubmitting it.


## Persistence & Concurrency

TradeRelay stores order state in MySQL 8.4, including execution status, broker identity, filled quantity, timestamps, and a monotonically increasing version.

Version-based optimistic concurrency control prevents stale writers from overwriting newer state:

```sql
UPDATE orders
SET status = ?, version = version + 1
WHERE id = ? AND version = ?;
```

If two processes read version `2`, only the first successful update can advance it to `3`. The second update no longer matches and is rejected as a version conflict.

Database constraints additionally enforce core invariants such as positive order quantities, valid sides and statuses, bounded filled quantities, unique internal order IDs, and unique broker order IDs.


## Testing

TradeRelay is tested across unit, concurrency, persistence, and integration layers.

- **Unit tests** validate signals, allocations, state transitions, execution, and reconciliation.
- **Concurrency tests** verify duplicate-order protection under concurrent submissions.
- **Race detection** uses Go's race detector to identify unsafe shared-memory access.
- **SQLMock tests** validate persistence behavior and optimistic concurrency logic.
- **MySQL integration tests** exercise execution, version conflicts, reconciliation, and startup recovery against a real MySQL 8.4 instance.
- **Failure-recovery tests** simulate crashes between broker acceptance and local persistence to verify safe recovery without blind resubmission.

```bash
go test ./...
go test -race ./...
```


## Current Limitations

TradeRelay is currently an experimental backend system rather than a production trading platform.

- Broker execution is simulated; no live trades are placed.
- MySQL is currently run locally for development and integration testing.
- Recovery is sequential rather than distributed across workers.
- A public REST API, authentication, observability, and real-time client updates are not yet implemented.
- Performance and throughput benchmarks have not yet been completed.
- The system does not claim exactly-once execution; reliability is built around stable order identity, idempotency, durable state, and reconciliation.


## Roadmap

The next development phase focuses on moving TradeRelay from a correctness-focused execution core toward a deployable backend service.

- REST API for signals, accounts, and orders
- Retry policies with exponential backoff and jitter
- Structured logging, metrics, and health/readiness endpoints
- Graceful shutdown and startup recovery hardening
- Authentication, authorization, and execution audit history
- Bounded concurrency, rate limiting, and backpressure
- Docker Compose development environment
- Latency, throughput, and failure-recovery benchmarks
- Real broker adapter and real-time order updates

  ---

## Tech Stack

- **Go** — execution engine, state management, concurrency, and broker abstraction
- **MySQL 8.4** — durable order persistence and optimistic concurrency control
- **Docker** — local MySQL environment
- **database/sql + go-sql-driver/mysql** — database access
- **SQLMock** — persistence unit testing
- **Go race detector** — shared-memory concurrency validation

---

## Running & Testing

Start the local MySQL instance:

```bash
docker run --name traderelay-mysql \
  -e MYSQL_ROOT_PASSWORD=traderelay \
  -e MYSQL_DATABASE=traderelay \
  -p 3306:3306 \
  -d mysql:8.4
```

Apply the database migration:

```bash
docker exec -i traderelay-mysql \
  mysql -uroot -ptraderelay traderelay \
  < migrations/001_create_orders.sql
```

Run the test suite:

```bash
go test ./...
go test -race ./...
```

Run MySQL integration tests:

```bash
TRADERELAY_MYSQL_DSN='root:traderelay@tcp(127.0.0.1:3306)/traderelay?parseTime=true' \
go test ./...
```

---

## Repository Structure

```text
trade-relay/
├── cmd/                 # Application entry points
├── internal/
│   ├── allocation/      # Multi-account trade allocation
│   ├── broker/          # Broker implementations
│   ├── execution/       # Execution, reconciliation, and recovery
│   ├── store/           # In-memory and MySQL persistence
│   └── trading/         # Signals, orders, validation, and state model
├── migrations/          # MySQL schema migrations
├── go.mod
└── README.md
```

The project is structured so trading-domain logic remains separate from persistence and broker-specific implementations, allowing execution behavior to be tested independently of external infrastructure.
