# Growing Pains

Use this document to record real ops workflow issues, tool-calling failures,
approval mistakes, and the fixes that made `hermes` safer and easier to trust.

## 2026-04-01

- Early pivots risked turning HERMES into a student-facing matchmaking surface.
  The fix was to re-lock HERMES as staff-only so APOLLO remains the member app.

## 2026-04-03

- Symptom: the first HERMES tracer could have drifted toward a richer "who is
  in the facility?" answer even though the stable public ATHENA read surface
  only exposes facility occupancy.
  Cause: the staff-facing framing was broader than the first trustworthy
  upstream data shape.
  Fix: narrow Tracer 8 to `hermes ask occupancy --facility <id>` and make the
  output explicitly source-backed from ATHENA's public occupancy endpoint.
  Rule: HERMES must not promise a richer staff answer than its public upstream
  truth can support.

- Symptom: the Tracer 8 hardening output looked noisy because several commands
  failed during destructive verification.
  Cause: the audit intentionally exercised missing-input, invalid-config,
  malformed-upstream, and unavailable-upstream paths, but the docs did not yet
  distinguish expected destructive failures from actual runtime regressions.
  Fix: document the expected failure paths in the runbook and explain the local
  truth versus deployed truth boundary in the README.
  Rule: HERMES hardening docs must label expected destructive failures
  explicitly so error output is interpreted as verification evidence, not as
  automatic product breakage.

- Symptom: runtime inspection of the occupancy slice still depended mostly on
  CLI output and upstream behavior rather than dedicated HERMES success-path
  logs.
  Cause: the first slice optimized for a narrow executable CLI boundary before
  adding richer operational observability.
  Fix: keep this as a documented non-blocking carry-forward gap instead of
  pretending the current observability is deeper than it is.
  Rule: future HERMES widening should add low-noise structured request/result
  logs before claiming stronger operational maturity.
