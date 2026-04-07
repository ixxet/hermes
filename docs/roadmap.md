# hermes Roadmap

## Objective

Keep HERMES as a staff-facing read-first runtime that widens only after the
source-backed boundary is trustworthy and observable.

## Current Line

Current active line: `v0.1.0`

- one read-only occupancy CLI question is real
- ATHENA public truth is the only upstream source in active use
- the slice is locally proven only
- richer observability, deployment proof, and write authority remain deferred

## Planned Release Lines

| Planned tag | Intended purpose | Restrictions | What it should not do yet |
| --- | --- | --- | --- |
| `v0.1.1` | observability hardening for Tracer 14 | keep the surface read-only and occupancy-only | do not widen into richer staff questions or writes |
| `v0.2.0` | live deployment proof for Milestone 1.7 | prove the existing occupancy slice in-cluster and stop there | do not imply write authority or broad assistant maturity |
| `v0.3.0` | one richer read-only staff question if a stable public upstream surface exists | keep the new question source-backed and narrow | do not invent identity-level answers without public upstream truth |
| `v0.4.0` | first write action plus approval boundary | add explicit write authority only with approval discipline | do not widen into broad workflow orchestration in the same line |

## Next Ladder Role

| Line | Role | Why it matters |
| --- | --- | --- |
| `Tracer 14` | observability-only hardening | makes the current occupancy slice operationally inspectable before any wider staff claim |
| `Milestone 1.7` | bounded live deployment proof | upgrades HERMES from local/runtime truth to deploy truth |
| `Tracer 17` | one richer read-only question | broadens the staff pillar only after observability and deployment trust exist |

## Boundaries

- keep staff operations sourced from public upstream service truth
- keep write actions later than observability hardening and live deployment proof
- do not widen into agent orchestration before the read boundary is trusted
- do not fabricate richer answers than upstream truth can support

## Tracer / Workstream Ownership

- `Tracer 8`: first read-only staff occupancy slice
- `Tracer 14`: HERMES observability hardening
- `Milestone 1.7`: live HERMES deployment proof
- later lines: richer read-only question, then first write action plus approval
