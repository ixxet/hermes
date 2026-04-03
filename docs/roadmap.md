# hermes Roadmap

## Objective

Add a staff-facing interface that can safely consume ATHENA data before it
starts taking real actions.

## First Implementation Slice

- read current occupancy from ATHENA
- answer one bounded staff CLI question:
  `hermes ask occupancy --facility <id>`
- keep the first slice narrow, observable, and read-only
- leave approval design and write actions deferred

## Boundaries

- no real booking writes in the first slice
- no member identity answer if the public ATHENA surface does not provide it
- no broad tool surface before the ATHENA dependency is proven stable
- no premature multi-agent complexity
- no deployment widening in this tracer

## Exit Criteria

- one staff question can be answered using real ATHENA-backed data
- the result shape is stable and identifies the source service
- the read path is traceable and easy to debug
- malformed, timeout, and unavailable upstream paths fail clearly
- the write path remains explicitly deferred

## Current State

Tracer 8 now proves the first real HERMES slice:

- `hermes ask occupancy --facility <id>` is real
- the command uses ATHENA's public
  `GET /api/v1/presence/count?facility=` surface directly
- the output is structured with `facility_id`, `current_count`,
  `observed_at`, and `source_service`
- the command supports `json` and `text` output without widening into a chat
  gateway
- missing facility input, invalid config, malformed upstream JSON, upstream
  timeouts, and upstream 500s all fail clearly
- the slice is locally proven only; deployed truth is unchanged

## Tracer Ownership

- `Tracer 8`: first read-only staff occupancy question over real ATHENA truth
