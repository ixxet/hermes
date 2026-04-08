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

## Versioning Discipline

HERMES now follows formal pre-`1.0.0` semantic versioning.

- `PATCH` releases cover observability, hardening, docs, deployment closeout,
  and bounded non-widening fixes
- `MINOR` releases cover new bounded read capabilities, new write
  capabilities, or intentional boundary changes
- pre-`1.0.0` breaking changes still require a `MINOR`, never a `PATCH`

## Planned Release Lines

| Planned tag | Intended purpose | Restrictions | What it should not do yet |
| --- | --- | --- | --- |
| `v0.1.1` | observability hardening for Tracer 14 | keep the surface read-only and occupancy-only | do not widen into richer staff questions or writes |
| `v0.1.2` | live deployment proof for Milestone 1.7 if runtime changes are required | prove the existing occupancy slice in-cluster and stop there | do not imply write authority or broad assistant maturity |
| deployment-only closeout | live deployment proof for Milestone 1.7 if runtime stays unchanged | keep the runtime line at `v0.1.1` and close deployment proof in companion repos/docs | do not overstate a new capability line when only deployment truth changed |
| `v0.2.0` | one richer read-only staff question if a stable public upstream surface exists | keep the new question source-backed and narrow | do not invent identity-level answers without public upstream truth |
| `v0.3.0` | first write action plus approval boundary | add explicit write authority only with approval discipline | do not widen into broad workflow orchestration in the same line |

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
- let `Milestone 1.7` close in companion deploy/docs repos if runtime widening
  is unnecessary

## Tracer / Workstream Ownership

- `Tracer 8`: first read-only staff occupancy slice
- `Tracer 14`: HERMES observability hardening
- `Milestone 1.7`: live HERMES deployment proof
- `Tracer 17`: one richer read-only question on `v0.2.0`
- later lines: first write action plus approval
