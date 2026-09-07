# TradeRelay
TradeRelay is a fault tolerant trade execution backend written in Go for coordinating a single trading decision across multiple brokerage accounts. The system is built around a failure that cannot be solved with a database transaction alone. A external broker may accept an order even when the application never receives or persists the acknowledgement. A timeout, process crash, or database failure can therefore leave local execution state uncertain while a real external side effect has already occurred. TradeRelay treats this uncertainty as a state recovery problem. It creates a durable execution plan before broker submission, assigns stable identities to intended orders, persists explicit execution state, and reconciles ambiguous outcomes against broker state before allowing execution to proceed.

The objective: 
A retry may repeat an operation, but it must never create a second logical execution for the same intended order. 

## The Problem

Trade execution crosses a consistency boundary that the application does not control.

TradeRelay can persist an intended order in MySQL and submit that order to a broker, but those operations cannot participate in the same atomic transaction. Once the broker accepts an order, rolling back local database state cannot reverse the external execution.

This creates an ambiguous failure window:

```text
TradeRelay                         Broker
    |                                |
    | persist SUBMITTING             |
    |                                |
    |------ submit order ----------->|
    |                                |
    |                         order accepted
    |                                |
    |<------ acknowledgement --------|
    |
    X  process fails before
       acknowledgement is persisted

## Hypothesis

A trade execution system can recover safely from failures that occur after an order may have reached the broker by preserving enough durable state to determine the order's actual outcome before deciding whether it should be submitted again.

## System Design

TradeRelay separates trade intent from external execution. A trading signal is first converted into an account-level execution plan, and the complete plan is persisted before any broker submission begins.

Each planned order receives a deterministic identity derived from the signal and target account. Once persisted, that plan becomes the authoritative execution intent for the signal; retries load the existing plan rather than recalculating it from potentially changed account state.

Execution then proceeds from durable state. Orders transition to `SUBMITTING` before crossing the broker boundary. If the outcome becomes uncertain, TradeRelay preserves that uncertainty and requires reconciliation with the broker rather than treating the order as safe to resubmit.

```text
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
Order Execution
      |
      +--------> Broker
      |            |
      |            v
      |       External State
      |            |
      v            |
Persistent State <-+
      ^
      |
Reconciliation / Recovery

## Execution Lifecycle

TradeRelay represents each order as an explicit state machine. State transitions define what the system knows about an order at each point in execution and determine whether external submission is safe.

```text
CREATED
   |
   v
SUBMITTING
   |
   +--------------------+
   |                    |
   v                    v
ACKNOWLEDGED         UNKNOWN
   |                    |
   v                    |  reconcile
PARTIALLY_FILLED        |
   |                    v
   v              authoritative
FILLED             broker state

REJECTED

**Important definitions for reference when following the project:**

CREATED: The order has been durably planned and persisted but has not yet been submitted to the broker.
SUBMITTING: Broker submission has begun. The broker may already have received or accepted the order.
UNKNOWN: The submission outcome is ambiguous. TradeRelay must query the broker before deciding what happened.
ACKNOWLEDGED: The broker confirms that the order exists and has been accepted for execution.
PARTIALLY FILLED: The broker reports that only part of the requested quantity has executed.
FILLED: The broker reports that the entire requested quantity has executed.
REJECTED: The broker rejected the order.
Client Order ID: stable identity TradeRelay uses across submission and recovery.
Broker Order ID: external identity assigned by the broker.
Signal ID: identifies the original trading decision.
Execution Plan: the persisted set of account-level orders created from that signal.


## Recovery and Reconciliation

A broker request can fail without revealing whether the trade itself failed. For example, TradeRelay may submit an order and lose the response after the broker has already accepted it. In that case, submitting the order again would be unsafe because the original order may already exist.

TradeRelay therefore follows a broker first recovery rule:

**When submission outcome is ambiguous, observe external broker state before deciding whether another submission is safe.**

For unresolved SUBMITTING and UNKNOWN orders, TradeRelay uses the order's stable identity to query the broker rather than immediately calling the submission path again.

Persisted Order
SUBMITTING / UNKNOWN
        |
        v
Does TradeRelay have
a Broker Order ID?
     /       \
   yes        no
    |          |
    v          v
Lookup by    Lookup by stable
Broker ID    Client Order ID
     \         /
      \       /
        v   v
   Broker State
        |
        v
Validate Result
        |
        v
Reconcile Local State
        |
        v
Persist Updated State

If a broker order ID is already known, reconciliation queries the broker using that identifier. If it is missing, TradeRelay queries using the stable client order ID instead. The returned broker state is validated before local state is changed. TradeRelay rejects inconsistent results such as impossible fill quantities, decreasing filled quantity, or a broker order identity that conflicts with the persisted order. Only after the external state has been observed and validated is the reconciled state persisted back to MySQL. This separates recovery from retry: uncertainty triggers observation first, not another external side effect.

## Persistence and Concurrency
TradeRelay uses MySQL as the authoritative store for execution state. Before broker submission begins, the complete account-level execution plan is persisted atomically.

Signal
   |
   v
Build Execution Plan
   |
   v
BEGIN TRANSACTION
   |
   +--> Persist Order A
   +--> Persist Order B
   +--> Persist Order C
   |
   v
COMMIT
   |
   v
Broker Execution May Begin

Note: If any order in the plan cannot be persisted, the transaction is rolled back. This prevents execution from beginning with only part of the intended plan durably recorded. Once a plan exists for a signal, subsequent processing loads the persisted plan rather than recalculating it. This prevents retries from changing the original execution intent if account configuration has changed since the signal was first processed.

Optimistic Concurrency

Each persisted order contains a version number. Updates are conditional on the version originally read by the caller.

UPDATE orders
SET status = ?, version = version + 1
WHERE id = ? AND version = ?;

If two processes attempt to update the same version of an order, only one can successfully advance it. A later stale update no longer matches the persisted version and is rejected as a version conflict.

This makes the database update an authoritative commit point for state transitions and prevents stale writers from silently overwriting newer execution state.

## Failure Model

TradeRelay assumes that the application, database, network, and broker can fail independently. Correctness therefore cannot depend on every submission receiving a definitive response or every process completing normally.

The system is designed around the following failure scenarios:

| Failure | Risk |
|---|---|
| **Process crash during submission** | The broker may accept an order before TradeRelay persists the result. |
| **Broker timeout** | TradeRelay cannot determine from the timeout alone whether the order was accepted. |
| **Lost broker response** | External execution may succeed while local state remains `SUBMITTING` or `UNKNOWN`. |
| **Duplicate signal or retry** | The same execution intent may enter the system more than once. |
| **Concurrent processing** | Multiple callers may attempt to act on the same persisted order state. |
| **Partial plan persistence** | Some account orders could become durable while others are lost before execution begins. |
| **Stale database update** | A process operating on an older order version could overwrite newer execution state. |
| **Inconsistent broker state** | A broker response may conflict with locally known identity, quantity, or fill progression. |

These failures are treated as expected operating conditions rather than exceptional cases outside the execution model. TradeRelay's recovery behavior is designed around preserving enough durable information to determine what happened after a failure, rather than assuming that an interrupted operation did not occur.

## Experimental Method

TradeRelay is evaluated by deliberately introducing failures at critical points in the execution lifecycle and verifying the resulting persisted state, broker state, and recovery behavior.

The tests focus on whether the system preserves execution identity and converges on the correct order state when normal execution is interrupted.

### Test Scenarios

| Scenario | What Is Tested |
|---|---|
| **Concurrent duplicate requests** | Whether repeated processing of the same execution intent creates more than one logical order. |
| **Plan persistence failure** | Whether broker execution is prevented when the complete execution plan cannot be persisted. |
| **Persisted-plan retry** | Whether a retry uses the original durable execution plan instead of recalculating execution intent. |
| **Ambiguous submission** | Whether `SUBMITTING` and `UNKNOWN` orders avoid blind resubmission. |
| **Lost broker acknowledgement** | Whether an order can be recovered by its stable client order ID when the broker order ID was never persisted. |
| **Stale concurrent update** | Whether optimistic concurrency rejects a writer operating on an outdated order version. |
| **Invalid broker state** | Whether inconsistent identities, fill quantities, or fill progression are rejected during reconciliation. |
| **Startup recovery** | Whether unresolved persisted orders can be discovered and reconciled after execution is interrupted. |

The implementation is tested at multiple boundaries using Go unit tests, concurrency tests, SQLMock persistence tests, MySQL integration tests, and the Go race detector.

```bash
go test ./...
go test -race ./...

The objective is not to demonstrate production trading performance. The experiments evaluate the narrower correctness hypothesis which is whether durable intent, stable identity, explicit uncertainty, and reconciliation allow interrupted execution to recover without treating an unknown outcome as safe to resubmit.

## Results

The current implementation demonstrates the core correctness properties required by the hypothesis under controlled failure and concurrency tests.

| Experiment | Observed Result |
|---|---|
| **Atomic plan persistence** | The complete execution plan committed when all order writes succeeded and rolled back when persistence failed before completion. |
| **Persisted-plan recovery** | A repeated execution attempt loaded the previously persisted plan rather than recalculating execution intent. |
| **Ambiguous submission recovery** | An `UNKNOWN` order without a broker order ID was reconciled through its stable client order ID without issuing another broker submission. |
| **State-aware retry** | Persisted `ACKNOWLEDGED`, `SUBMITTING`, and `UNKNOWN` orders were not blindly resubmitted when the execution plan was processed again. |
| **Optimistic concurrency** | Version-conditional persistence rejected stale order updates after the persisted version had advanced. |
| **Recovery discovery** | Persisted `SUBMITTING` and `UNKNOWN` orders were discovered and passed through reconciliation. |
| **Broker-state validation** | Invalid fill quantities, fill regression, inconsistent status/fill combinations, and conflicting broker identities were rejected during reconciliation. |
| **Concurrent duplicate identity** | Concurrent attempts using the same logical identity converged on a single stored identity in the tested duplicate-submission path. |

These results support the hypothesis that durable execution intent, stable identity, explicit uncertainty, and broker first reconciliation can preserve logical execution identity across the failure scenarios tested.
They do not establish exactly once execution, production scale fault tolerance, or production trading performance. Those properties are outside the scope of the current experiments.

## Repository Structure

TradeRelay separates trading-domain logic, execution coordination, broker behavior, and persistence into distinct packages.

```text
trade-relay/
├── cmd/
│   └── traderelay/       # Application entry point and service startup
├── internal/
│   ├── allocation/       # Signal-to-account allocation
│   ├── api/              # HTTP service boundary
│   ├── broker/           # Broker implementations and simulation
│   ├── execution/        # Planning, submission, reconciliation, and recovery
│   ├── store/            # MySQL and in-memory persistence
│   └── trading/          # Core order, signal, account, and state models
├── migrations/           # MySQL schema
├── go.mod
└── README.md

## Running and Testing

TradeRelay requires Go and a running MySQL instance with the project migration applied.

Set the MySQL connection string and start the service:

```bash
export TRADERELAY_DB_DSN='root@tcp(127.0.0.1:3307)/trade_relay?parseTime=true'
go run ./cmd/traderelay
```

Verify that the service is running:

```bash
curl http://localhost:8080/healthz
```

Expected response:

```json
{"status":"ok"}
```

Run the test suite:

```bash
go test ./...
go test -race ./...
```

## Future Work

The next phase of TradeRelay will focus on strengthening the correctness claims under more adversarial execution conditions.

The immediate priority is validating concurrent first execution races and multiple processes attempting to create, load, and advance the same execution plan while competing for the same persisted order state. Additional failure injection will target crashes at each persistence and broker boundary to verify that recovery converges without unsafe resubmission.

Later experiments will extend the broker simulator with delayed and out-of-order responses, partial fills, broker unavailability, and controlled network failures. Performance evaluation will then measure execution latency, recovery time, throughput, and behavior under sustained concurrent load.

A real broker adapter would be introduced only after these correctness properties are more extensively validated against the simulated failure model.

## Current Limitations

TradeRelay is an experimental execution backend designed to evaluate correctness and recovery behavior rather than production trading performance. Broker execution is currently modeled through a stateful simulator, and durable execution state is maintained through a single MySQL instance. The system has not been validated against a live brokerage API, distributed database topology, or real money execution environment.

The current experiments establish specific recovery and persistence properties, but they do not establish exactly once execution. In particular, stronger end-to-end testing is still required for simultaneous execution attempts where multiple processes compete to advance the same order toward broker submission. Optimistic concurrency protects persisted state in this path, but the complete broker-execution race has not yet been demonstrated under all relevant interleavings.

Recovery is also currently sequential, and the system has not been evaluated under sustained load. Throughput, latency, saturation behavior, distributed recovery, database failover, and production-scale broker behavior therefore remain outside the scope of the current results.

