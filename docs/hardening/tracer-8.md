# Tracer 8 Hardening

This document is the standalone hardening artifact for Tracer 8.

It records what was actually audited, which failures were expected destructive
checks, what remains unproven, and why Tracer 8 still closed cleanly.

## Scope

Tracer 8 audits one bounded HERMES question only:

```bash
hermes ask occupancy --facility <id>
```

That runtime reads ATHENA's public:

```text
GET /api/v1/presence/count?facility=<id>
```

This document does not claim:

- a broad assistant runtime
- write actions
- approvals
- gateway behavior
- live in-cluster HERMES deployment
- identity-level roster answers

## Audited Claim

Tracer 8 claims:

- HERMES can answer one real bounded staff question
- the answer is sourced from upstream public truth
- the runtime is read-only
- failure handling is explicit
- deployment truth is unchanged

## Commit Under Audit

- `hermes`: `60ec3ead578a6a05d3887b8681721ee8532051d5`
- control-plane closeout at the time of tracer closure:
  - `ashton-platform`: `0573d3b95918b03141360f2d80d6e4b690f179bc`

This hardening document was added afterward to preserve the audit as a single
GitHub-facing artifact.

## Exact Commands Rerun

```bash
git -C /Users/zizo/Personal-Projects/ASHTON/hermes status --short
git -C /Users/zizo/Personal-Projects/ASHTON/hermes branch --show-current
git -C /Users/zizo/Personal-Projects/ASHTON/hermes rev-parse HEAD
git -C /Users/zizo/Personal-Projects/ASHTON/hermes rev-parse @{u}
git -C /Users/zizo/Personal-Projects/ASHTON/hermes merge-base @ @{u}

git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform status --short
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform branch --show-current
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform rev-parse HEAD
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform rev-parse @{u}
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform merge-base @ @{u}

git -C /Users/zizo/Personal-Projects/ASHTON/athena status --short
git -C /Users/zizo/Personal-Projects/ASHTON/athena branch --show-current
git -C /Users/zizo/Personal-Projects/ASHTON/athena rev-parse HEAD
git -C /Users/zizo/Personal-Projects/ASHTON/athena rev-parse @{u}
git -C /Users/zizo/Personal-Projects/ASHTON/athena merge-base @ @{u}
```

```bash
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test ./...
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test -count=2 ./...
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test -count=10 ./internal/athena -run '^(TestClientCurrentOccupancyConsumesAthenaReadSurface|TestClientCurrentOccupancyMapsUpstreamFailuresClearly)$'
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test -count=10 ./internal/command -run '^(TestAskOccupancyCommandRequiresFacility|TestAskOccupancyCommandOutputsStableJSONShape|TestAskOccupancyCommandSupportsTextOutputAndClearErrors)$'
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go build ./cmd/hermes

cd /Users/zizo/Personal-Projects/ASHTON/athena && go test ./internal/server ./internal/presence -run '^(TestHealthEndpoint|TestCurrentOccupancy|TestCurrentOccupancyFiltersByFacility|TestCurrentOccupancyReturnsZeroForMissingFacility)$'
```

## Smoke Evidence

Real upstream runtime:

```bash
cd /Users/zizo/Personal-Projects/ASHTON/athena
ATHENA_HTTP_ADDR='127.0.0.1:18090' go run ./cmd/athena serve
curl -sS -i http://127.0.0.1:18090/api/v1/health
curl -sS 'http://127.0.0.1:18090/api/v1/presence/count?facility=ashtonbee'
```

Successful HERMES read:

```bash
cd /Users/zizo/Personal-Projects/ASHTON/hermes
go run ./cmd/hermes ask occupancy \
  --facility ashtonbee \
  --athena-base-url http://127.0.0.1:18090 \
  --format json
```

Observed result:

```json
{"facility_id":"ashtonbee","current_count":9,"observed_at":"2026-04-03T07:27:53Z","source_service":"athena"}
```

Unknown facility:

```bash
go run ./cmd/hermes ask occupancy \
  --facility missing \
  --athena-base-url http://127.0.0.1:18090 \
  --format json
```

Observed result:

```json
{"facility_id":"missing","current_count":0,"observed_at":"2026-04-03T07:29:07Z","source_service":"athena"}
```

## Expected Destructive Failures

These failures were intentionally exercised and are part of the proof:

Missing required facility:

```bash
go run ./cmd/hermes ask occupancy --athena-base-url http://127.0.0.1:18090 --format json
```

Observed:

```text
ERROR: required flag(s) "facility" not set
```

Invalid timeout override:

```bash
go run ./cmd/hermes ask occupancy --facility ashtonbee --athena-base-url http://127.0.0.1:18090 --timeout -1s --format json
```

Observed:

```text
ERROR: http timeout must be greater than zero
```

Unavailable upstream:

```bash
go run ./cmd/hermes ask occupancy --facility ashtonbee --athena-base-url http://127.0.0.1:18091 --format json
```

Observed:

```text
ERROR: athena occupancy request failed: Get "http://127.0.0.1:18091/api/v1/presence/count?facility=ashtonbee": dial tcp 127.0.0.1:18091: connect: connection refused
```

Malformed upstream response:

```text
ERROR: athena occupancy response is malformed: unexpected EOF
```

These are expected audit outcomes. They are not evidence that the happy path is
broken.

## Verified Truth

- verified local truth only
- HERMES answers one bounded occupancy question from a real ATHENA runtime
- HERMES uses ATHENA's public HTTP surface, not a private DB path
- HERMES stays read-only
- HERMES output identifies `source_service = "athena"`
- repeated reads against unchanged ATHENA state stayed consistent

## Unverified Truth

- no live HERMES deployment proof
- no gateway, chat, or agent orchestration proof
- no identity-level facility roster answer
- no broader staff operations surface

## Carry-Forward Gaps

- HERMES success-path observability is still thin
- deployment truth is still unchanged from earlier milestones
- richer staff questions should only be added against stable public upstream
  surfaces

## Final Verdict

Tracer 8 is closure-clean with carry-forward gaps.

That verdict is based on:

- one real bounded staff read path
- explicit failure handling
- a clean read-only boundary
- no overclaim of deployment truth
