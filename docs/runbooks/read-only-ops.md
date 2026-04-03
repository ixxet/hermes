# HERMES Read-Only Ops Runbook

## Purpose

Use this runbook for the first HERMES slice before any write paths exist.

## Rules

- read ATHENA before writing anything anywhere
- keep staff tooling separate from member-facing flows
- approval design can exist before approval execution
- use public upstream service surfaces, not private database access
- identify the source service in the output
- fail clearly on malformed or unavailable upstream data

## Required Checks

- one staff-facing question resolves against real ATHENA-backed data
- the path from prompt to tool result is easy to trace
- no write action is exposed accidentally
- `go test ./...`
- `go test -count=2 ./...`
- repeated boundary runs for `./internal/athena` and `./internal/command`
- `go build ./cmd/hermes`

## Tracer 8 Verified Commands

```bash
cd /Users/zizo/Personal-Projects/ASHTON/hermes
go test ./...
go test -count=2 ./...
go test -count=10 ./internal/athena -run '^(TestClientCurrentOccupancyConsumesAthenaReadSurface|TestClientCurrentOccupancyMapsUpstreamFailuresClearly)$'
go test -count=10 ./internal/command -run '^(TestAskOccupancyCommandRequiresFacility|TestAskOccupancyCommandOutputsStableJSONShape|TestAskOccupancyCommandSupportsTextOutputAndClearErrors)$'
go build ./cmd/hermes
```

## Tracer 8 Verified Smoke

Start a real ATHENA runtime first:

```bash
cd /Users/zizo/Personal-Projects/ASHTON/athena
ATHENA_HTTP_ADDR='127.0.0.1:18090' go run ./cmd/athena serve
```

Successful read-only smoke:

```bash
curl -sS http://127.0.0.1:18090/api/v1/health
curl -sS 'http://127.0.0.1:18090/api/v1/presence/count?facility=ashtonbee'

cd /Users/zizo/Personal-Projects/ASHTON/hermes
go run ./cmd/hermes ask occupancy \
  --facility ashtonbee \
  --athena-base-url http://127.0.0.1:18090 \
  --format json

go run ./cmd/hermes ask occupancy \
  --facility missing \
  --athena-base-url http://127.0.0.1:18090 \
  --format json
```

Failure-mode smoke:

```bash
cd /Users/zizo/Personal-Projects/ASHTON/hermes
go run ./cmd/hermes ask occupancy \
  --athena-base-url http://127.0.0.1:18090 \
  --format json

go run ./cmd/hermes ask occupancy \
  --facility ashtonbee \
  --athena-base-url http://127.0.0.1:18090 \
  --timeout -1s \
  --format json

go run ./cmd/hermes ask occupancy \
  --facility ashtonbee \
  --athena-base-url http://127.0.0.1:18091 \
  --format json
```

Expected smoke outcomes:

- HERMES returns the same `facility_id`, `current_count`, and `observed_at`
  shape that ATHENA exposes, plus `source_service = "athena"`
- unknown facilities remain source-backed and return `current_count = 0` if
  ATHENA says so
- missing `--facility` fails clearly before an upstream request is attempted
- invalid `--timeout` fails clearly during config validation
- unavailable upstream reads fail clearly and return a non-zero CLI exit code
- malformed upstream reads fail clearly and return a non-zero CLI exit code
- no write behavior exists in the tracer

## Hardening Interpretation

Do not treat every failing command in the hardening pass as a product bug.

- expected destructive failures are evidence that the CLI rejects bad input and
  bad upstream states explicitly
- unexpected failures are only the cases where a valid occupancy read against a
  healthy ATHENA runtime breaks
- Prometheus outage does not affect this tracer unless a future HERMES tracer
  widens deployment truth
