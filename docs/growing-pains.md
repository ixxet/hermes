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
