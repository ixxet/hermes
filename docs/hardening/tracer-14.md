# Tracer 14 Hardening

This document is the standalone hardening artifact for Tracer 14.

It records the bounded observability claim, the repeated proof that was rerun,
what was smoke-tested directly, what remains unproven, and why the line closed
cleanly without widening the runtime.

## Scope

Tracer 14 hardens one bounded HERMES question only:

```bash
hermes ask occupancy --facility <id>
```

That runtime still reads ATHENA's public:

```text
GET /api/v1/presence/count?facility=<id>
```

This document does not claim:

- a richer staff question
- write actions
- approvals
- gateway behavior
- Prometheus or deployment work
- live in-cluster HERMES deployment
- identity-level roster answers

## Audited Claim

Tracer 14 claims:

- HERMES emits low-noise structured request/result/outcome observability for
  the existing occupancy slice
- stdout answer payloads remain unchanged
- the occupancy runtime remains read-only and occupancy-only
- success and failure paths stay explainable and repeatable
- deployment truth is unchanged

## Implementation Commits Under Audit

- runtime change in `hermes`:
  - `2619a88c50ec89a9ff9d5cd47a8df97d1bd781ac`
- control-plane closeout sync in `ashton-platform`:
  - `9ae6139024b2df03785d124b5fed957bf7f52d88`

This hardening artifact and the release tags were added afterward so the proof
is preserved in GitHub rather than left only in terminal history.

## Exact Commands Rerun

```bash
git -C /Users/zizo/Personal-Projects/ASHTON/hermes status -sb
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform status -sb

git -C /Users/zizo/Personal-Projects/ASHTON/hermes log --oneline --decorate -n 5
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform log --oneline --decorate -n 5

git -C /Users/zizo/Personal-Projects/ASHTON/hermes show --stat --summary --decorate 2619a88
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform show --stat --summary --decorate 9ae6139

git -C /Users/zizo/Personal-Projects/ASHTON/hermes tag -l 'v0.1.1'
git -C /Users/zizo/Personal-Projects/ASHTON/hermes ls-remote --tags origin 'refs/tags/v0.1.1'
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform tag -l 'v0.0.20'
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform ls-remote --tags origin 'refs/tags/v0.0.20'
```

```bash
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test ./internal/command -count=10
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test ./internal/ops -count=10
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test ./internal/athena -count=10

cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test ./internal/command -run '^(TestAskOccupancyCommandRequiresFacility|TestAskOccupancyCommandOutputsStableJSONShapeAndStructuredSuccessLog|TestAskOccupancyCommandStructuredFailureLogs|TestAskOccupancyCommandBlankFacilityStaysValidationOnly|TestAskOccupancyCommandObservabilityRemainsLowNoiseAcrossRepeatedRuns|TestAskOccupancyCommandSupportsTextOutput)$' -count=10
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test ./internal/athena -run '^(TestClientCurrentOccupancyMapsUpstreamFailuresClearly|TestClientCurrentOccupancyClassifiesTransportFailure)$' -count=10
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test ./internal/config -run '^(TestConfigWithOverridesValidatesRequiredValues|TestConfigWithOverridesRejectsInvalidValues|TestLoadRejectsInvalidEnvTimeout)$' -count=10

cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test ./...
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go vet ./...
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go build ./cmd/hermes
```

## Direct Smoke Commands

A bounded local ATHENA-like stub on `127.0.0.1:18093` served deterministic
success, upstream-500, malformed, and slow responses for smoke verification.

Successful HERMES read with stdout and stderr separated:

```bash
tmpdir=$(mktemp -d)
cd /Users/zizo/Personal-Projects/ASHTON/hermes
go run ./cmd/hermes ask occupancy \
  --facility ashtonbee \
  --athena-base-url http://127.0.0.1:18093 \
  --format json \
  1>"$tmpdir/stdout.txt" \
  2>"$tmpdir/stderr.txt"
cat "$tmpdir/stdout.txt"
cat "$tmpdir/stderr.txt"
```

Failing HERMES read with stdout and stderr separated:

```bash
tmpdir=$(mktemp -d)
cd /Users/zizo/Personal-Projects/ASHTON/hermes
go run ./cmd/hermes ask occupancy \
  --facility boom \
  --athena-base-url http://127.0.0.1:18093 \
  --format json \
  1>"$tmpdir/stdout.txt" \
  2>"$tmpdir/stderr.txt"
cat "$tmpdir/stdout.txt"
cat "$tmpdir/stderr.txt"
```

In addition to those two smoke commands, a temporary deterministic CLI harness
repeated six cases three times each: success, upstream `500`, malformed
payload, timeout, config failure, and validation failure.

## Destructive Evidence

- repeated `./internal/command` runs stayed stable at `-count=10`
- repeated `./internal/athena` upstream failure mapping stayed stable at
  `-count=10`
- repeated `./internal/config` validation checks stayed stable at `-count=10`
- the direct CLI harness passed `6 cases x 3 runs`
- each direct CLI invocation emitted exactly:
  - one `request-start`
  - one `request-complete` or one `request-failed`
- stderr carried the structured observability lines
- stdout stayed clean:
  - success path kept the same JSON answer shape
  - failure paths kept stdout empty

## Smoke Evidence

Successful read observed:

```json
{"facility_id":"ashtonbee","current_count":9,"observed_at":"2026-04-02T16:00:00Z","source_service":"athena"}
```

Successful stderr observed:

```json
{"event":"request-start","component":"hermes","tracer":14,"question":"occupancy","request_id":"occ-000001","facility":"ashtonbee","upstream":"athena","outcome":"started","version":"dev"}
{"event":"request-complete","component":"hermes","tracer":14,"question":"occupancy","request_id":"occ-000001","facility":"ashtonbee","upstream":"athena","outcome":"success","duration_ms":2,"version":"dev","occupancy_count":9}
```

Failing stderr observed:

```text
{"event":"request-start","component":"hermes","tracer":14,"question":"occupancy","request_id":"occ-000001","facility":"boom","upstream":"athena","outcome":"started","version":"dev"}
{"event":"request-failed","component":"hermes","tracer":14,"question":"occupancy","request_id":"occ-000001","facility":"boom","upstream":"athena","outcome":"failed","duration_ms":2,"version":"dev","upstream_status":500,"error_kind":"upstream_error","error":"athena occupancy request failed with status 500: read path unavailable"}
ERROR: athena occupancy request failed with status 500: read path unavailable
exit status 1
```

That smoke proof confirms the user-facing answer shape stayed on stdout while
the structured observability stayed on stderr.

## Verified Truth

- verified local/runtime truth only
- HERMES still answers one bounded occupancy question only
- HERMES still reads ATHENA's public HTTP surface only
- HERMES stays read-only
- HERMES now emits low-noise structured request/result/outcome logs for that
  occupancy slice
- no stdout pollution from observability was introduced

## Verified Deployed Truth

- unchanged by Tracer 14 itself
- tagging `hermes v0.1.1` and `ashton-platform v0.0.20` does not mean
  `Milestone 1.7` is done

## Deferred Truth

- no live HERMES deployment proof
- no richer staff question beyond occupancy
- no gateway, chat, or agent orchestration proof
- no identity-level roster answer
- no write behavior

## Final Verdict

Tracer 14 is closure-clean and tagged.

That verdict is based on:

- bounded observability hardening only
- repeated destructive proof without flakiness
- direct smoke proof with separated stdout and stderr
- aligned repo-local and control-plane docs
- honest local/runtime versus deployed truth split
