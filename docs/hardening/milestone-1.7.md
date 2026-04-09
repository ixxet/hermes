# Milestone 1.7 Hardening

## Scope

Milestone 1.7 closes the existing HERMES occupancy slice as a bounded live
deployment proof. It does not add a new question, a public service, write
authority, or any Tracer 17 implication.

## Verified Truth

- HERMES still answers one read-only occupancy question only
- the deployed shape is an internal-only bounded runner deployment in
  `agents`
- the live path is exec-driven rather than publicly exposed
- ATHENA remains the upstream truth source
- no Go runtime patch was needed
- the runtime line remains `v0.1.1`; deployment truth moved separately

## Packaging And Build Truth

The repo now has a container packaging path at [`Dockerfile`](../../Dockerfile).
That path exists so the already-real CLI runtime can be packaged for deployment;
it does not add broader runtime capability.
The exact deployed image digest is pinned in the Prometheus deployment manifest
and deployment runbook.

## Deployment Shape

The cluster proof used one internal runner deployment:

- namespace: `agents`
- workload kind: `Deployment`
- replicas: `1`
- public service: none
- ingress: none
- write path: none

The runner stayed alive only so `kubectl exec` could invoke the real CLI
against live ATHENA-backed truth.

## Commands Used

Local runtime and packaging checks:

```bash
cd /Users/zizo/Personal-Projects/ASHTON/hermes
go test ./...
go vet ./...
go build ./cmd/hermes
docker buildx build --platform linux/amd64 --load -t hermes:milestone-1-7-local .
docker run --rm hermes:milestone-1-7-local version
```

Cluster proof:

```bash
kubectl rollout status -n agents deployment/hermes --timeout=180s
kubectl exec -n agents deploy/hermes -- /bin/sh -lc '/usr/local/bin/hermes ask occupancy --facility ashtonbee --format json 2>/proc/1/fd/2'
kubectl exec -n agents deploy/hermes -- /bin/sh -lc '/usr/local/bin/hermes ask occupancy --facility ashtonbee --format json 2>/proc/1/fd/2'
kubectl exec -n agents deploy/hermes -- /bin/sh -lc '/usr/local/bin/hermes ask occupancy --facility ashtonbee --format json 2>/proc/1/fd/2'
```

Safe negative proof:

```bash
kubectl exec -n agents deploy/hermes -- /bin/sh -lc '/usr/local/bin/hermes ask occupancy --facility "" --format json'
kubectl exec -n agents deploy/hermes -- /bin/sh -lc '/usr/local/bin/hermes ask occupancy --facility not-a-real-facility --format json'
```

Additional isolated negative probes were run in a local harness for bad
upstream URL, timeout, and upstream `500` behavior.

## Deferred Truth

- no public HERMES service was added
- no write authority was added
- no broader assistant runtime was added
- no Tracer 17 work was started
- no HERMES runtime patch was required for this milestone

## Closeout

Milestone 1.7 is deployment-only closeout truth. Companion deploy/docs repos
carry the cluster-facing shape, while HERMES remains occupancy-only and
read-only at `v0.1.1` runtime truth.
