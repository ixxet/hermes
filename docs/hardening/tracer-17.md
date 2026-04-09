# Tracer 17 Hardening

This document is the standalone hardening artifact for Tracer 17.

It records the bounded reconciliation claim, the exact proof rerun, what was
verified directly, what remained deferred, and why the line stays read-only.

## Scope

Tracer 17 hardens one richer HERMES question:

```bash
hermes ask reconciliation --facility <id> --window <duration> --bin <duration>
```

That runtime still reads upstream truth only:

- `GET /api/v1/presence/count?facility=<id>`
- `GET /api/v1/presence/history?facility=<id>&since=<rfc3339>&until=<rfc3339>`

This document does not claim:

- overrides
- write actions
- approval flows
- gateway behavior
- frontend or demo work
- deployment proof for the new reconciliation slice
- identity-level roster answers

## Audited Claim

Tracer 17 claims:

- HERMES now answers one richer read-only reconciliation question over stable
  ATHENA truth
- the answer includes a deterministic occupancy report and heat-map-style
  buckets over a bounded history window
- the answer remains privacy-safe and does not leak raw account values,
  resolved names, or hashed identities
- the runtime remains read-only
- deployment truth is unchanged from the earlier occupancy-only runner slice

## Exact Commands Rerun

```bash
git -C /Users/zizo/Personal-Projects/ASHTON/hermes status -sb
git -C /Users/zizo/Personal-Projects/ASHTON/athena status -sb
git -C /Users/zizo/Personal-Projects/ASHTON/ashton-platform status -sb

cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test ./...
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go test -count=5 ./internal/...
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go vet ./...
cd /Users/zizo/Personal-Projects/ASHTON/hermes && go build ./cmd/hermes

cd /Users/zizo/Personal-Projects/ASHTON/athena && go test ./...
cd /Users/zizo/Personal-Projects/ASHTON/athena && go test -count=5 ./internal/edgehistory ./internal/server
cd /Users/zizo/Personal-Projects/ASHTON/athena && go vet ./...
cd /Users/zizo/Personal-Projects/ASHTON/athena && go build ./cmd/athena
```

## Direct CLI Smoke

A deterministic local HTTP stub exposed:

- `GET /api/v1/presence/count`
- `GET /api/v1/presence/history`

Verified reconciliation success twice against the same upstream payload:

```bash
cd /Users/zizo/Personal-Projects/ASHTON/hermes
./hermes ask reconciliation \
  --facility ashtonbee \
  --window 2h \
  --bin 1h \
  --athena-base-url http://127.0.0.1:18095 \
  --format json
```

Verified destructive failures:

```bash
cd /Users/zizo/Personal-Projects/ASHTON/hermes
./hermes ask reconciliation \
  --facility ashtonbee \
  --window 1h \
  --bin 2h \
  --athena-base-url http://127.0.0.1:18095 \
  --format json

./hermes ask reconciliation \
  --facility disabled \
  --window 2h \
  --bin 1h \
  --athena-base-url http://127.0.0.1:18095 \
  --format json

./hermes ask reconciliation \
  --facility malformed \
  --window 2h \
  --bin 1h \
  --athena-base-url http://127.0.0.1:18095 \
  --format json
```

## Destructive Evidence

- repeated `go test -count=5 ./internal/...` stayed stable in HERMES
- repeated `go test -count=5 ./internal/edgehistory ./internal/server` stayed
  stable in ATHENA
- the richer reconciliation stdout payload stayed identical across repeated
  runs against the same upstream inputs
- each reconciliation invocation emitted exactly one `request-start` log and
  one `request-complete` or `request-failed` log
- invalid `--window` / `--bin` combinations failed clearly with no stdout
- upstream history-disabled responses failed clearly with `upstream_status=503`
- malformed history payloads failed clearly with `error_kind="decode_error"`

## Verified Truth

- verified local/runtime truth:
  - HERMES now answers one richer read-only reconciliation question
  - occupancy reports and heat-map-style buckets are deterministic for the same
    stable inputs
  - the richer read stays privacy-safe and does not leak raw IDs, names, or
    hashed identities
  - ATHENA now exposes one bounded privacy-safe history read surface to support
    HERMES without private DB or file shortcuts
- verified deployed truth:
  - unchanged
  - the earlier occupancy-only HERMES runner deployment remains the only
    deployed HERMES claim
- deferred truth:
  - no deployed proof for the new reconciliation slice
  - no write authority
  - no gateway, chat, or broad assistant UX
  - no public service expansion

## Final Verdict

Tracer 17 is closure-clean if release-line and tag discipline are handled
honestly at closeout.
