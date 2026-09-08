# Trade Relay

Trade Relay is a failure aware trade execution backend written in Go and MySQL. It coordinates a trading decision across brokerage accounts while preserving execution identity through retries, concurrent processing, ambiguous broker responses, and process interruption.

The central problem is that local database state and external broker execution cannot be committed atomically. A broker may accept an order even if Trade Relay times out or crashes before recording the acknowledgement. Trade Relay addresses this by persisting execution intent before external submission, using optimistic concurrency to establish a single submission winner, and reconciling uncertain outcomes against broker state before permitting further action.

The project focuses on one question: How can an execution system recover safely when it cannot immediately determine whether an external side effect occurred?

## Failure Boundary

A database transaction can protect Trade Relay's local state, but it cannot include the broker.

```text
Trade Relay                         Broker
     |                                |
     | persist SUBMITTING             |
     |                                |
     |------ SubmitOrder ------------>|
     |                                |
     |                         order accepted
     |                                |
     |<------ acknowledgement --------|
     |                                |
     X process crashes
```

At this point, the database may contain `SUBMITTING` while the broker already contains a live order.

Blindly retrying `SubmitOrder` would be unsafe.

Trade Relay therefore represents uncertainty explicitly and recovers by observing broker state rather than assuming that an interrupted request failed.

## Architecture


                    Trading Signal
                          |
                          v
                  Account Allocation
                          |
                          v
                   Execution Plan
                          |
                          v
              Atomic Plan Persistence
                          |
                          v
                  Order Coordinator
                          |
              CREATED -> SUBMITTING
                          |
                  version-checked
                     DB update
                          |
                          v
                       Broker
                          |
                  external state
                          |
             +------------+------------+
             |                         |
             v                         v
       definitive result         ambiguous result
             |                         |
             v                         v
      persist new state             UNKNOWN
                                       |
                                       v
                              Recovery Service
                                       |
                                       v
                              Broker Reconciliation
                                       |
                                       v
                              Persist Recovered State


Trade Relay separates execution intent from external execution. A signal is first converted into an account level execution plan. The complete plan is persisted before any order is allowed to reach the broker. Once persisted, that plan becomes authoritative. A retry loads the existing plan rather than recalculating it from potentially changed account configuration.

## Execution Lifecycle

Orders move through explicit states describing what Trade Relay knows about the external execution:

```text
CREATED
   |
   v
SUBMITTING
   |
   +----------------------+
   |                      |
   v                      v
ACKNOWLEDGED           UNKNOWN
   |                      |
   v                      | reconcile
PARTIALLY_FILLED          |
   |                      v
   v                 broker state
FILLED

REJECTED

Terminology to follow throughout the project:

CREATED — execution intent is durably persisted but broker submission has not begun.
SUBMITTING — the worker has claimed the submission transition and broker interaction may be in progress.
UNKNOWN — the broker outcome is ambiguous. Another submission is not assumed to be safe.
ACKNOWLEDGED — the broker confirms that the order exists.
PARTIALLY_FILLED / FILLED — execution progress reported by the broker.
REJECTED — submission was definitively rejected.

## Concurrent Submission Control

The important concurrency boundary is placed **before the external broker call**.

Each order contains a monotonically increasing version. A state update succeeds only if the persisted version still matches the version originally read by the caller.

Conceptually:

sql
UPDATE orders
SET
    status = ?,
    version = version + 1
WHERE id = ?
  AND version = ?;


Consider two workers that independently load the same order:


MySQL
Order A
status  = CREATED
version = 1

        Worker 1                 Worker 2
           |                        |
           | read version 1         | read version 1
           |                        |
           v                        v
     claim SUBMITTING         claim SUBMITTING
           |                        |
           +----------+-------------+
                      |
                      v
                 MySQL CAS
                  /       \
              succeeds    conflict
                 |           |
                 v           X
            SubmitOrder    STOP
                 |
                 v
               Broker


Only the worker that successfully persists CREATED → SUBMITTING may cross the broker boundary. A MySQL integration test creates two independent copies of the same persisted version and releases both submitters concurrently. The broker implementation is instrumented to count actual SubmitOrder calls.

The tested result is:

successful submitters:      1
version conflicts:          1
broker SubmitOrder calls:   1
final persisted state:      ACKNOWLEDGED
final persisted version:    3

Broker-side deduplication alone could hide duplicate calls. The test measures the external submission boundary directly.

## Durable Execution Plans

A trading signal may produce orders for multiple brokerage accounts.

Trade Relay persists the complete execution plan before allowing execution to begin:


Signal
  |
  v
Build Account-Level Orders
  |
  v
BEGIN TRANSACTION
  |
  +--> Order A
  +--> Order B
  +--> Order C
  |
  v
COMMIT
  |
  v
Broker execution may begin


If any plan write fails, the transaction rolls back. This prevents Trade Relay from executing a partially persisted plan where some intended account orders exist externally while the complete execution intent was never durably recorded. Persisted plans are also reused on retry. Changes to account configuration after plan creation therefore do not silently change an execution decision that has already become durable.

## Ambiguous Failure Recovery

A failed broker request does not necessarily mean the broker rejected or never received the order.

For example:


SubmitOrder()
      |
      v
Broker accepts order
      |
      X response lost


Trade Relay cannot safely infer:


request failed == trade did not happen


Instead, the order remains unresolved and recovery follows a **broker-first rule**:

> When an external outcome is uncertain, observe authoritative broker state before deciding what should happen next.

For `SUBMITTING` and `UNKNOWN` orders:

```text
Persisted unresolved order
           |
           v
   BrokerOrderID known?
       /          \
     yes           no
      |             |
      v             v
 lookup by      lookup by stable
 broker ID      client order ID
       \           /
        \         /
         v       v
        Broker State
             |
             v
       Validate Result
             |
             v
      Reconcile State
             |
             v
        Persist Update
```

If the broker order ID was never persisted, Trade Relay can locate the external order using the stable client-order identity. Recovery therefore does not begin by blindly calling `SubmitOrder` again.

## Broker-State Validation

Broker state is treated as authoritative for external execution, but it is not copied into local state without validation.

Reconciliation rejects results such as:

* a broker order identity that conflicts with the persisted order
* filled quantity greater than requested quantity
* filled quantity moving backward
* status and fill combinations that cannot both be true

Only validated broker state is allowed to advance persisted execution state.

## Recovery Service

Trade Relay can scan durable storage for unresolved orders:


MySQL
 |
 +--> SUBMITTING
 |
 +--> UNKNOWN
 |
 v
Recovery Service
 |
 v
Reconciler
 |
 v
Broker
 |
 v
Recovered Local State


This allows an interrupted process to reconstruct unresolved execution from durable state rather than depending on the memory of the process that originally submitted the order. The recovery path has been integration-tested against MySQL with multiple persisted order states.

## Failure Model

Trade Relay assumes that the application, database, network, and broker can fail independently.

| Failure                                | System behavior                                                                         |
| -------------------------------------- | --------------------------------------------------------------------------------------- |
| Concurrent submission                  | Version-conditional transition selects one submission winner before the broker boundary |
| Stale writer                           | MySQL rejects the update with a version conflict                                        |
| Plan persistence failure               | Transaction rolls back before broker execution begins                                   |
| Process interruption                   | Durable order state remains available for recovery                                      |
| Broker timeout                         | Outcome is treated as ambiguous rather than assumed unsuccessful                        |
| Lost acknowledgement                   | Broker state can be recovered using stable order identity                               |
| Duplicate processing                   | Existing durable execution intent is reused                                             |
| Changed account configuration          | Persisted plan remains authoritative for the original signal                            |
| Invalid broker state                   | Reconciliation rejects inconsistent external state                                      |
| `SUBMITTING` / `UNKNOWN` after restart | Recovery service discovers and reconciles unresolved orders                             |

## Tested Properties

Trade Relay is tested at the domain, persistence, concurrency, and recovery boundaries.

The current suite includes:

* order lifecycle and transition tests
* allocation tests
* execution-plan tests
* plan-executor tests
* simulated broker tests
* reconciliation tests
* broker-first recovery tests
* SQLMock persistence tests
* real MySQL integration tests
* concurrent submission integration testing
* stale-version conflict testing
* recovery scanning
* HTTP API tests
* Go race-detector execution

Run the complete suite with:

```bash
go test ./...
go test -race ./...
```

Both suites currently pass.

## What the Tests Demonstrate

The current implementation demonstrates several specific correctness properties under the tested failure model:

### 1. Atomic intent before execution

A multi-account execution plan must become durable before broker submission begins. Persistence failure prevents execution from proceeding with only part of the plan recorded.

### 2. Persisted plan reuse

Once a plan exists, retry processing loads that plan instead of silently recalculating execution intent.

### 3. Stale write rejection

A caller operating on an outdated order version cannot overwrite a newer persisted state.

### 4. Single winner submission coordination

Two callers starting from the same persisted order version do not both cross the external broker boundary. The tested race produces one successful state claim, one version conflict, and one broker submission invocation.

### 5. Explicit uncertainty

Ambiguous submission outcomes are represented as unresolved state rather than being interpreted as safe failures.

### 6. Broker-first recovery

An interrupted order can be rediscovered from durable storage and reconciled against broker state, including by stable client-order identity when the broker order ID was never persisted locally.

### 7. Reconciliation invariants

Impossible fill progression and conflicting broker identities are rejected rather than silently incorporated into local execution state.

## Correctness Boundary

Trade Relay does **not** claim universal exactly-once execution.

A local database cannot atomically commit an external brokerage side effect. Instead, the system combines:


durable execution intent
        +
stable order identity
        +
single-winner state transitions
        +
broker-side identity
        +
explicit uncertain states
        +
broker-first reconciliation


to make retries and recovery safer across the failure scenarios represented by the current implementation and tests.

the project demonstrates specific failure-handling invariants rather than claiming an end-to-end guarantee that the architecture cannot prove.

## Repository Structure

```text
trade-relay/
├── cmd/
│   └── traderelay/       # service entry point
├── internal/
│   ├── allocation/       # signal-to-account allocation
│   ├── api/              # HTTP boundary
│   ├── broker/           # broker interface and simulator
│   ├── execution/        # planning, submission, reconciliation, recovery
│   ├── store/            # MySQL persistence
│   └── trading/          # domain models and order lifecycle
├── migrations/           # MySQL schema
├── go.mod
└── README.md
```

## Running Locally

Trade Relay requires Go and MySQL with the project schema applied.

Set the application database connection:

```bash
export TRADERELAY_DB_DSN='root@unix(/tmp/mysql.sock)/trade_relay?parseTime=true'
```

Start the service:

```bash
go run ./cmd/traderelay
```

Verify the HTTP service:

```bash
curl http://localhost:8080/healthz
```

Expected response:

```json
{"status":"ok"}
```

MySQL integration tests use:

```bash
export TRADERELAY_MYSQL_DSN='root@unix(/tmp/mysql.sock)/trade_relay?parseTime=true'
```

Then:

```bash
go test ./...
go test -race ./...
```

The exact DSN will vary by local MySQL configuration.

## Current Scope and Limitations

Trade Relay is an experimental backend for studying correctness at the boundary between durable local state and external trade execution. The current implementation uses a deterministic broker simulator and a single MySQL instance. It has not been validated against a live brokerage API, distributed database topology, or real money execution environment. The concurrency tests establish the tested single-winner submission invariant for callers competing from the same persisted version. They do not prove correctness under every possible distributed-system interleaving or infrastructure failure. Recovery currently runs sequentially, and the project has not been benchmarked for production throughput, latency, saturation behavior, database failover, or large scale recovery. Authentication, authorization, secrets management, trading risk controls, production observability, and operational safeguards required for a real-money system are also outside the current scope.

