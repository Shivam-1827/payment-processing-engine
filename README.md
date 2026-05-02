# Deterministic Distributed Payment Engine

A distributed payment processing system built with Go, C++, Kafka-compatible messaging, Redis, and PostgreSQL. The repository is organized around a deterministic event flow: an API gateway accepts payment intents, a C++ sequencer processes and reorders them in memory, an orchestrator settles completed payments against a double-entry ledger, and a webhook dispatcher forwards results to merchants.

## Overview

The goal of the project is to model a low-latency payment pipeline that separates concerns across services:

- `services/api-gateway` validates requests, enforces authentication and idempotency, and publishes payment intents to Kafka.
- `services/sequencer` consumes intents, performs deterministic in-memory processing, and publishes validated events.
- `services/orchestrator` consumes validated events, calls an external bank abstraction, and records the settlement in PostgreSQL.
- `services/webhook-service` reads the outbox table and delivers webhook notifications to merchants.

The infrastructure layer in `infra/` contains PostgreSQL migrations, Grafana, and Prometheus assets. The root `docker-compose.yml` wires the stack together for local development.

## Architecture

```text
[Client]
	 -> [Go API Gateway]
	 -> [Redis idempotency lock]
	 -> [Kafka / Redpanda: payment_intents]
	 -> [C++ Sequencer]
	 -> [Kafka / Redpanda: payment_validated]
	 -> [Go Orchestrator]
	 -> [External bank client]
	 -> [PostgreSQL double-entry ledger]
	 -> [Outbox table]
	 -> [Webhook dispatcher]
	 -> [Merchant webhook endpoint]
```

## Tech Stack

- Go 1.25
- C++20
- PostgreSQL 15
- Redis 7
- Redpanda / Kafka-compatible messaging
- `pgx` for PostgreSQL access
- `go-chi` for HTTP routing
- `librdkafka` via the C++ sequencer wrapper

## Repository Layout

```text
services/
	api-gateway/      HTTP entrypoint for payment intent submission
	orchestrator/      Kafka consumer that settles payments and writes ledger entries
	sequencer/         C++ event sequencer and Kafka bridge
	webhook-service/   Outbox-based webhook delivery worker
infra/
	migrations/        PostgreSQL schema migrations
	grafana/           Dashboard assets
	prometheus/        Metrics configuration
libs/
	proto/
	shared-go/
```

## Local Development

### Prerequisites

- Docker and Docker Compose
- Go 1.25 or newer if you want to run services outside containers
- A C++20 toolchain if you want to build the sequencer locally

### Start the Infrastructure

From the repository root:

```bash
docker compose up --build
```

This starts PostgreSQL, Redis, Redpanda, the migration job, and the application services defined in `docker-compose.yml`.

### Run Go Services Locally

If you want to run a service outside Docker, export the required environment variables first. The services use these defaults when variables are not provided:

- `PORT` for the API gateway, default `8080`
- `REDIS_ADDR` for the API gateway, default `localhost:6379`
- `KAFKA_BROKERS`, default `localhost:9092`
- `KAFKA_TOPIC`, default `payment_intents`
- `KAFKA_TOPIC_VALIDATED`, default `payment_validated`
- `KAFKA_GROUP_ID`, default `go_orchestrator_group`
- `DB_URL`, default `postgres://ledger_admin:supersecretpassword@localhost:5432/payment_engine?sslmode=disable`
- `MERCHANT_WEBHOOK_URL`, default `https://httpbin.org/post`

Example:

```bash
export KAFKA_BROKERS=localhost:9092
export REDIS_ADDR=localhost:6379
export DB_URL=postgres://ledger_admin:supersecretpassword@localhost:5432/payment_engine?sslmode=disable

go run ./services/api-gateway/cmd
go run ./services/orchestrator/cmd
go run ./services/webhook-service/cmd
```

### Build the Sequencer

The sequencer lives in `services/sequencer` and uses CMake. A typical local build looks like this:

```bash
cmake -S services/sequencer -B services/sequencer/build
cmake --build services/sequencer/build
```

## API Gateway

The gateway exposes a single payment submission route:

- `POST /v1/payments`

Required headers:

- `Authorization` for API key authentication
- `Idempotency-Key` for distributed deduplication

Example payload:

```json
{
  "from_account": "11111111-1111-1111-1111-111111111111",
  "to_account": "22222222-2222-2222-2222-222222222222",
  "amount": 1250,
  "currency": "usd"
}
```

Successful requests return `202 Accepted` with a JSON body indicating that the payment intent was queued.

## Data Flow

1. The API gateway validates the request body and idempotency key.
2. A Redis-backed lock prevents duplicate concurrent submissions for the same idempotency key.
3. The payment intent is published to Kafka and acknowledged immediately.
4. The C++ sequencer consumes the intent, processes it deterministically in memory, and emits a validated event.
5. The orchestrator consumes the validated event, calls the bank abstraction, and persists ledger entries in PostgreSQL.
6. The webhook worker reads unsent outbox rows and delivers merchant notifications.

## Testing

Run the Go test suite from the repository root:

```bash
go test ./... -v
```

If Go reports missing `go.sum` entries for transitive `pgx` dependencies, run:

```bash
go mod tidy
```

## Notes

- The project uses structured JSON logging in the Go services.
- PostgreSQL migrations live in `infra/migrations` and are applied through the `db-migrations` container in the compose stack.
- The webhook service uses an outbox polling pattern with `FOR UPDATE SKIP LOCKED` semantics in PostgreSQL.

## License

No license file is currently included in the repository.
