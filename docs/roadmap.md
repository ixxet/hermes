# hermes Roadmap

## Objective

Add a staff-facing interface that can safely consume ATHENA data before it starts taking real actions.

## First Implementation Slice

- read current occupancy from ATHENA
- answer one natural-language facility status query
- define the approval model for future write operations
- keep the first slice narrow and observable

## Boundaries

- no real booking writes in the first slice
- no broad tool surface before the ATHENA dependency is proven stable
- no premature multi-agent complexity

## Exit Criteria

- one staff question can be answered using real ATHENA-backed data
- the read path is traceable and easy to debug
- the write path remains explicitly deferred behind approval rules
