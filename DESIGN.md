# Design

This document describes the architecture of `gin-vagrant-demo` and the reasoning behind
the choices made. The project is intentionally small; its purpose is to demonstrate a
clean path from Go source to a containerized service running inside a reproducible VM,
not to be a full application.

## Goals

- Show a Gin HTTP service deployed the "real" way: built as a Docker image, run as a
  container, on a machine provisioned from scratch.
- Make the whole lifecycle reproducible and one-command (`vm.sh`).
- Keep the runtime image small, secure, and free of unnecessary tooling.
- Demonstrate production-shaped patterns (graceful shutdown, health checks, build
  metadata) without growing into a framework.

## Non-goals

- Persistence, databases, or external service integration.
- Authentication / authorization.
- Horizontal scaling, orchestration (Kubernetes, Swarm), or multi-host deployment.
- TLS termination (assumed to be handled by an upstream proxy if ever needed).

## System overview

There are three layers, each isolated from the next:

```
Host (developer machine)
│  vm.sh  ──drives──▶  Vagrant / VirtualBox
│
└── Guest VM (Ubuntu 24.04)
    │  Ansible (ansible_local) installs Go + Docker CE
    │  scripts/*.sh build + test inside the VM
    │
    └── Docker container (distroless)
        └── gin-server  ──listens on :8080──▶  forwarded to host :8080
```

The host never needs Go, Docker, or Ansible. Everything is installed and runs inside the
VM, so the only host dependencies are Vagrant and VirtualBox. This keeps the demo
reproducible regardless of what the developer happens to have installed.

## Components

### Application (`cmd/gin-demo`, `internal/server`)

The application is split into a thin entrypoint and a router package:

- `internal/server` owns route registration and handlers and exposes
  `NewRouterWithInfo(BuildInfo)`. It has no knowledge of process lifecycle, ports, or
  signals. This keeps it trivially testable with `httptest` — the tests build a router
  and serve requests against it directly, no network involved.
- `cmd/gin-demo/main.go` owns everything environmental: reading `PORT`, constructing the
  `http.Server`, signal handling, graceful shutdown, and the healthcheck mode.

`NewRouter()` is retained as a zero-argument convenience (used by tests and any caller
that does not care about build info); it delegates to `NewRouterWithInfo` with
placeholder values.

### Routing

Four behaviors, all returning JSON:

| Route        | Purpose                                                              |
| ------------ | ------------------------------------------------------------------- |
| `/`          | Liveness sanity / demo greeting.                                    |
| `/health`    | Health probe — used by `vm.sh test` and the Docker `HEALTHCHECK`.   |
| `/version`   | Returns the build metadata baked in at compile time.               |
| `NoRoute`    | Uniform JSON `404` instead of Gin's default plain-text response.    |

A JSON 404 is a small but deliberate choice: a JSON API should never hand a client an
HTML/plain-text error for an unknown path, so all responses — including errors — share
one content type.

### Build metadata

`main.version`, `main.commit`, and `main.buildTime` are package-level variables set at
link time via `-ldflags -X`. They default to `dev`/`none`/`unknown` for a plain
`go build` or `go run`, and are populated from git in `scripts/build_and_deploy.sh` /
the Docker build. The values are surfaced at runtime through `/version`.

This is the standard Go approach to stamping a binary: no code generation, no embedded
files, and the running service can always report exactly which commit it came from.

## Key design decisions

### Graceful shutdown over `r.Run()`

Gin's `r.Run()` is convenient but offers no way to drain in-flight requests. The
entrypoint instead constructs an explicit `http.Server`, runs `ListenAndServe` in a
goroutine, and blocks on a `SIGINT`/`SIGTERM` channel. On signal it calls
`srv.Shutdown(ctx)` with a 10s timeout so active requests complete before the process
exits. This is the behavior a container runtime expects when it sends `SIGTERM` to stop
a container, and it avoids dropping requests on redeploy.

### Self-probing healthcheck (`-healthcheck` flag)

The same binary, when invoked with `-healthcheck`, makes an HTTP request to its own
`/health` on `127.0.0.1:$PORT` and maps the result to an exit code. The Docker
`HEALTHCHECK` runs `gin-server -healthcheck`.

The motivation is the runtime image (see below): a distroless image has no shell and no
`curl`/`wget`, so the conventional `HEALTHCHECK CMD curl ...` is impossible. Putting the
probe inside the binary keeps the image minimal while still giving Docker a real health
signal. It also means the health logic lives in one place and is testable.

### Distroless, non-root runtime image

The Dockerfile is multi-stage:

1. **Builder** (`golang:1.22`): copies `go.mod` + `go.sum` first and runs
   `go mod download` so the dependency layer is cached independently of source changes,
   then builds a fully static binary (`CGO_ENABLED=0`) with `-trimpath` and stripped
   symbols (`-s -w`).
2. **Runtime** (`gcr.io/distroless/static-debian12:nonroot`): contains only the binary
   plus CA certs and a non-root user. No shell, no package manager, smaller attack
   surface, and runs as non-root by default.

A static binary is what makes the distroless `static` base viable — there is nothing to
dynamically link against. The earlier `debian:stable-slim` runtime worked but shipped an
entire userland that the service never uses.

`go mod download` (cached) is used instead of `go mod tidy` at build time: `tidy` mutates
`go.mod`/`go.sum` and is a development-time operation, not something a reproducible build
should do.

### Vagrant + Ansible provisioning

The VM uses `ansible_local`, so the playbook runs *inside* the guest and the host needs
no Ansible. `provisioning/site.yml` installs Go from apt and Docker CE from Docker's
official apt repository (adding the GPG key and repo), enables the Docker service, and
adds the `vagrant` user to the `docker` group. It also sets up a usable shell
environment (bash completion, a fixed prompt) for when a human SSHes in to poke around.

Provisioning is declarative and idempotent, so `vagrant provision` can be re-run safely.

### `vm.sh` as the single entrypoint

All host-side actions go through `vm.sh`, which wraps `vagrant` and `vagrant ssh -c`. The
build and test logic that must run inside the VM lives in `scripts/build_and_deploy.sh`
and `scripts/run_tests.sh`; `vm.sh` simply invokes them over SSH. This keeps the
host/guest boundary explicit and means the in-VM scripts can also be run by hand.

## Configuration

| Variable   | Default | Effect                          |
| ---------- | ------- | ------------------------------- |
| `PORT`     | `8080`  | Listen port (and healthcheck target). |
| `GIN_MODE` | (debug) | Set to `release` to quiet Gin's logging in production. |

Configuration is environment-driven only; there are no config files. For a demo of this
size, env vars are sufficient and match how the value would be supplied to a container.

## Testing

Tests live in `internal/server` and exercise each route through `httptest` against the
router returned by `NewRouter`/`NewRouterWithInfo`, asserting both status code and exact
JSON body. Because the router has no lifecycle dependencies, the tests are fast and
hermetic. `scripts/run_tests.sh` additionally runs `go vet` and enables the race detector
(`go test -race`).

## From demo to product

The demo's value as a starting point is its *seams*: the router has no lifecycle
dependencies, the entrypoint owns all environmental concerns, configuration is already
externalized, and the image is already minimal and non-root. That structure means the
gap below can be closed incrementally — each capability slots into an existing seam
rather than forcing a rewrite. What's missing is not architecture, it's surface area.

This section is a roadmap, ordered so that each phase is shippable on its own and earlier
phases de-risk later ones. The guiding principle: make it *correct and safe* before
making it *featureful*, and make it *observable* before you scale it.

### What already carries over

These are production-shaped and become the foundation rather than throwaway scaffolding:

- Clean entrypoint/router split (testable handlers, lifecycle in one place).
- Graceful shutdown on `SIGTERM` — correct behavior under any container orchestrator.
- Build-metadata stamping and `/version` — every running instance is traceable to a commit.
- Distroless, non-root, static image — a small attack surface to build on.
- A single host-side driver (`vm.sh`) — the pattern for a richer task runner / `Makefile`.

### Current → product, at a glance

| Concern        | Demo today                          | Product target                                              |
| -------------- | ----------------------------------- | ----------------------------------------------------------- |
| Logging        | Gin's default text logger           | Structured `slog` (JSON) + request IDs + log levels         |
| Health         | single `/health`                    | `/livez` (process) **and** `/readyz` (dependencies)         |
| Config         | ad-hoc `os.Getenv`                  | Typed, validated config struct; secrets from a manager      |
| API            | 4 fixed demo routes                 | Versioned (`/v1`) domain resources + OpenAPI spec           |
| State          | none                                | Postgres + migrations + repository layer; Redis cache       |
| Security       | none                                | AuthN/AuthZ, TLS, rate limiting, security headers, CORS     |
| Resilience     | graceful shutdown only              | Server timeouts, limits, downstream retries/circuit breakers|
| Observability  | logs only                           | Prometheus metrics + OpenTelemetry traces + dashboards      |
| Delivery       | local `scripts/*.sh`                | CI/CD: lint, test, scan, sign, multi-arch push              |
| Runtime        | one container in one VM             | Kubernetes/Helm (or PaaS); VM stays as the dev sandbox      |

### Phase 0 — Harden (low effort, high value, no new dependencies)

Do this first. It's mostly a day or two of work and closes real gaps that exist *today*.

- **HTTP server timeouts.** `cmd/gin-demo/main.go` constructs `&http.Server{Addr, Handler}`
  with no timeouts, which leaves the service exposed to slow-client (Slowloris) resource
  exhaustion. Set `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`.
  This is the single highest-value one-line-each change in the whole roadmap.
- **Structured logging.** Replace `gin.Default()`'s text logger with a `slog` (JSON)
  logger plus a request-ID middleware that injects a correlation ID into the context and
  echoes it on an `X-Request-ID` response header. Keep `gin.Recovery()`.
- **Liveness vs. readiness.** Split the current `/health` into `/livez` (is the process
  up?) and `/readyz` (are dependencies — DB, cache — reachable?). Orchestrators need both:
  failing readiness pulls an instance out of rotation without killing it.
- **Typed configuration.** Replace scattered `os.Getenv` with one validated config struct
  (e.g. `caarlos0/env` or `koanf`), parsed and validated at startup so misconfiguration
  fails fast and loudly instead of at first request.
- **Edge middleware.** Rate limiting, request body size limits, CORS, and security headers
  (`Strict-Transport-Security`, `X-Content-Type-Options`, etc.).
- **Quality gates in the test script.** Add `golangci-lint` and `govulncheck` to
  `scripts/run_tests.sh` alongside the existing `go vet` / `go test -race`.

### Phase 1 — Become a real API

- **Versioned domain routes.** Group real resources under `/v1/...` so the API can evolve
  without breaking clients. Standardize on one JSON error envelope (the existing 404 shape
  is the seed) for *every* non-2xx response.
- **Request validation.** Bind and validate request bodies (Gin already pulls in
  `go-playground/validator`); reject malformed input with structured field-level errors.
- **OpenAPI contract.** Author or generate an OpenAPI 3 spec and serve interactive docs.
  A contract enables generated clients and contract tests.
- **Persistence.** Add Postgres via `pgx` with a connection pool, schema migrations
  (`goose` or `golang-migrate`, run as an init step), and a thin repository layer behind
  an interface so handlers stay testable. Add Redis where caching or rate-limit state
  needs to be shared across instances.

### Phase 2 — Secure it

- **Authentication.** OIDC/JWT for end users; API keys or mTLS for service-to-service.
- **Authorization.** Role- or attribute-based access control enforced in middleware.
- **TLS & secrets.** Terminate TLS at an ingress/reverse proxy (or in-process with
  autocert); pull secrets from a manager (Vault, AWS/GCP Secrets Manager, or Kubernetes
  Secrets) rather than plain env vars.
- **Audit logging.** Record security-relevant events (authn, authz denials, mutations)
  on a separate, tamper-evident stream.

### Phase 3 — Operate it

- **Metrics.** A `/metrics` endpoint exposing RED metrics (request **R**ate, **E**rror
  rate, request **D**uration) via a Gin Prometheus middleware, plus Go runtime metrics.
- **Tracing.** OpenTelemetry with an OTLP exporter; propagate the request ID into spans
  so logs, metrics, and traces correlate.
- **CI/CD.** A pipeline (e.g. GitHub Actions) that runs lint + race tests + `govulncheck`,
  builds **multi-arch** images (the Dockerfile currently hard-codes `GOARCH=amd64`),
  generates an SBOM, signs the image (`cosign`), and pushes on tagged releases
  (`goreleaser` for changelogs/versioning).
- **Runtime platform.** Graduate from one container in one VM to Kubernetes + a Helm chart
  (or a managed PaaS) for horizontal scaling, rolling/canary deploys, and self-healing.
  Keep the Vagrant flow as the zero-host-dependency local sandbox — it remains the fastest
  way to reproduce the full stack on a laptop.
- **Test depth.** Integration tests against a real Postgres via `testcontainers`, and load
  tests (`k6`/`vegeta`) with a tracked latency/throughput budget.

### Phase 4 — Productize (only if it's a SaaS)

If the goal is a multi-customer product rather than an internal service:

- Multi-tenancy with strict per-tenant data isolation.
- Usage quotas / rate plans, metering, and billing integration.
- A self-serve onboarding flow and an admin console.

### Sequencing note

Phases 0 and 3's observability are the load-bearing ones: you cannot safely add
persistence, auth, or scale without timeouts, structured logs, readiness probes, and
metrics already in place. Resist starting at Phase 1 (features) before Phase 0 (safety) is
done — it's the difference between a demo that grew and a product that was built.
