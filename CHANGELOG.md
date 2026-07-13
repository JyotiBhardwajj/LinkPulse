# Changelog

All notable changes to **LinkPulse** are documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [Unreleased]

### Planned
- Web dashboard UI
- Link analytics export (CSV/JSON)
- Custom branded domains
- QR code generation per link

---

## [0.11.0] – 2026-07-13

### Added – Day 11: Final Production Hardening, Security Audit & Architecture Polish
- `SecurityHeaders` middleware: injects `X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Content-Security-Policy`, `Permissions-Policy`, and `Cross-Origin-Resource-Policy` on every response.
- Benchmark suite (`internal/benchmark/`) with `BenchmarkJWTGeneration`, `BenchmarkJWTValidation`, `BenchmarkJSONMarshalLink`, `BenchmarkWorkerSubmit`, `BenchmarkResolveShortCode`, `BenchmarkCreateLink`, and `BenchmarkCacheLookup`.
- MIT License, CHANGELOG, GitHub Issue and PR templates.
- Security middleware unit tests.

---

## [0.10.0] – 2026-07-13

### Added – Day 10: Production Readiness & Reliability Layer
- Kubernetes-aligned health probes: `/health/live`, `/health/ready`, `/health/startup`.
- Parallel dependency health checks with per-checker timeouts.
- Critical vs. optional dependency classification for `/health/ready` semantics.
- Atomic readiness state tracking (`ReadinessState`) with instant-false on shutdown.
- Context-aware worker pool with `Shutdown(ctx)` draining guarantee.
- Graceful HTTP server shutdown via `http.Server.Shutdown(ctx)`.
- Docker health check pointing to `/health/live`.
- Kubernetes probe configuration in `docs/production.md`.
- `RecordHealthCheckDuration`, `RecordReadinessState`, `RecordStartupDuration` Prometheus metrics.
- Lifecycle configuration (`HEALTH_TIMEOUT`, `STARTUP_TIMEOUT`, `SHUTDOWN_TIMEOUT`, `DATABASE_TIMEOUT`, `REDIS_TIMEOUT`).

---

## [0.9.0] – 2026-07-11

### Added – Day 9: Developer Experience, Standardized APIs & OpenAPI
- API versioning via `/api/v1` and `/api/v2` (placeholder).
- RFC 7807 `application/problem+json` error responses.
- Correlation / request IDs propagated to all responses via `X-Request-ID`.
- ETag support on GET endpoints with `If-None-Match` / `304 Not Modified`.
- `Cache-Control` headers on GET responses.
- Deprecation middleware with `Sunset` and `Deprecation` RFC headers.
- OpenAPI 3.0 specification (`docs/swagger.json`) with reusable schemas, examples, and `operationId`s.
- Pagination metadata (`total`, `page`, `page_size`, `total_pages`, `has_next`, `has_prev`).
- Validation error normalization: sorted, JSON-field-named, human-readable messages.

---

## [0.8.0] – 2026-07-10

### Added – Day 8: Observability, Prometheus Metrics & Grafana
- Prometheus metrics integration via `internal/metrics/prometheus.go`.
- Metrics middleware recording latency, status codes, request counts per route.
- Worker metrics: queue size, processed events, dropped events, active workers.
- Cache metrics: hits, misses, errors per operation.
- GORM database metrics plugin.
- `/metrics` endpoint serving Prometheus text format.
- Grafana dashboard provisioning with datasource and dashboard JSON.
- `docker-compose.yml` Prometheus + Grafana service definitions.

---

## [0.7.0] – 2026-07-09

### Added – Day 7: Advanced Analytics
- Click analytics recording with browser, OS, device, country, city detection.
- `GET /api/v1/links/:id/analytics` returning time-series click data.
- `GET /api/v1/analytics/overview` aggregated dashboard endpoint.
- `GET /api/v1/analytics/clicks` click-over-time query with range filters.
- Analytics worker pool for non-blocking background event processing.

---

## [0.6.0] – 2026-07-08

### Added – Day 6: Rate Limiting & Request Timeouts
- Per-IP token-bucket rate limiting middleware.
- Request timeout middleware with configurable duration.
- `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` headers.
- `429 Too Many Requests` with RFC 7807 problem body.

---

## [0.5.0] – 2026-07-07

### Added – Day 5: Redis Caching & Audit Logger
- Redis-backed link cache with TTL, JSON serialization, and singleflight stampede prevention.
- Structured audit logger with async queue and guaranteed drain on shutdown.
- `Exists` cache check to avoid redundant database lookups on redirect resolution.

---

## [0.4.0] – 2026-07-06

### Added – Day 4: Authentication, JWT & Refresh Token Rotation
- User registration and login with bcrypt password hashing.
- JWT access tokens (HS256, 15-minute TTL).
- Refresh Token Rotation (RTR) with SHA-256 hashing for storage.
- Session management: list, single logout, logout-all.
- `Auth` middleware extracting claims from `Authorization: Bearer` header.
- RBAC middleware for role-based endpoint protection.

---

## [0.3.0] – 2026-07-05

### Added – Day 3: Link CRUD & Redirect
- `POST /api/v1/links` – create short link with optional custom code and expiry.
- `GET /api/v1/links` – paginated list of user's links.
- `GET /api/v1/links/:id` – fetch single link.
- `PATCH /api/v1/links/:id` – update long URL, active state, or expiry.
- `DELETE /api/v1/links/:id` – soft delete a link.
- `GET /r/:code` – high-performance redirect with click recording.
- Short code uniqueness retry loop (configurable attempts).

---

## [0.2.0] – 2026-07-04

### Added – Day 2: Clean Architecture & Repository Layer
- Clean Architecture directory layout: `cmd`, `internal/{handler,service,repository,models,cache,worker,middleware,routes,config,logger,metrics,health}`.
- GORM PostgreSQL repository implementations for `link`, `user`, `refresh_token`, `analytics`.
- Database migrations via `golang-migrate`.
- Structured `slog` logging with JSON and text formatters.

---

## [0.1.0] – 2026-07-03

### Added – Day 1: Project Bootstrap
- Go module initialization (`linkpulse`).
- Gin HTTP framework setup.
- Configuration loading via Viper + `.env`.
- PostgreSQL connection pool via GORM.
- Redis connection client.
- GitHub Actions CI pipeline: `go vet`, `go test -race`, `go build`, `gofmt` check.
- Docker Compose for local development services.
