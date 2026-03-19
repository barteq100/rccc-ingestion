# rccc-ingestion

Job ingestion pipeline for the Remote Career Command Center MVP.

## Purpose

`rccc-ingestion` is responsible for:

- source adapters for external job providers
- normalization into the canonical job payload
- obvious duplicate detection
- scheduler and retry behavior
- idempotent upsert calls into `rccc-api`

## Tech Choice

- Go for straightforward workers and parsers
- REST integration into `rccc-api` to keep the ownership boundary explicit

## Initial Structure

- `cmd/worker`: process entrypoint
- `internal/sources`: source adapters
- `internal/normalize`: canonical DTO shaping
- `internal/dedupe`: deduplication heuristics
- `internal/scheduler`: polling and retry orchestration
- `internal/client`: API integration boundary
- `testdata`: source payload fixtures

## Local Development

1. Copy `.env.example` to `.env`.
2. Run `docker compose up --build`.
3. For local Go execution, run `go run ./cmd/worker`.

The current scaffold only logs a startup placeholder.

## Assumptions

- MVP will start with Greenhouse and Lever.
- Ingestion owns source-specific parsing and dedupe heuristics.
- API owns canonical persistence and final validation.

## MVP Backlog

- define canonical ingestion payload contract
- implement Greenhouse adapter
- implement Lever adapter
- implement normalization rules
- implement duplicate detection
- implement retry and backoff policy
- add fixture-driven parser tests
