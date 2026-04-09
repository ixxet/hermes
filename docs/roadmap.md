# hermes Roadmap

## Objective

Keep HERMES as a staff-facing read-first runtime that widens only after the
source-backed boundary is trustworthy and observable.

## Current Line

Current active line: `v0.2.0` repo truth

- one read-only occupancy CLI question is real
- one richer read-only reconciliation CLI question is now real
- low-noise structured request/result/outcome observability is now real for
  both HERMES questions
- ATHENA public truth now includes the existing occupancy read plus one bounded
  privacy-safe stable-history read used only for reconciliation
- the reconciliation slice is locally/runtime proven
- deployment proof remains unchanged and still applies only to the earlier
  occupancy runner slice

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
| `v0.1.2` | reserved for a future runtime-safe patch if HERMES code ever changes | prove the existing occupancy slice in-cluster and stop there | do not imply write authority or broad assistant maturity |
| deployment-only closeout | Milestone 1.7 live deployment proof with unchanged runtime | keep the runtime line at `v0.1.1` and close deployment proof in companion repos/docs | do not overstate a new capability line when only deployment truth changed |
| `v0.2.0` | one richer read-only reconciliation question over current + stable-history truth | keep the new question source-backed, privacy-safe, and narrow | do not invent identity-level answers or broaden into multiple operator shells |
| `v0.3.0` | first write action plus approval boundary | add explicit write authority only with approval discipline | do not widen into broad workflow orchestration in the same line |

## Next Ladder Role

| Line | Role | Why it matters |
| --- | --- | --- |
| `Milestone 1.7` | bounded live deployment proof | upgraded HERMES from local/runtime truth to deploy truth for the occupancy slice |
| `Tracer 17` / `v0.2.0` | one richer read-only reconciliation question | broadens the staff pillar with stable-history reports and heat-map-style reads while staying read-only |

## Boundaries

- keep staff operations sourced from public upstream service truth
- keep write actions later than observability hardening and live deployment proof
- do not widen into agent orchestration before the read boundary is trusted
- do not fabricate richer answers than upstream truth can support
- keep stable-history widening bounded to one reconciliation question rather
  than a menu of unrelated reads
- let `Milestone 1.7` close in companion deploy/docs repos if runtime widening
  is unnecessary
- keep the deployed HERMES shape internal-only and exec-driven, not a public
  service

## Tracer / Workstream Ownership

- `Tracer 8`: first read-only staff occupancy slice
- `Tracer 14`: HERMES observability hardening
- `Milestone 1.7`: live HERMES deployment proof
- `Tracer 17`: one richer read-only reconciliation question on `v0.2.0`
- later lines: first write action plus approval
