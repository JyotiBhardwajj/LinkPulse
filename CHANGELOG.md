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

## [1.0.0] - 2026-07-14

### Added
- Comprehensive Grafana k6 load testing suite supporting smoke, baseline, stress, spike, and soak test scenarios.
- Integrated Swagger / OpenAPI 3.0 specification served directly from memory.
- Dynamic automated SQL migrations executed during application startup.
- Full Railway deployment configurations with Redis authenticated clustering.

### Fixed
- Relaxed Content-Security-Policy (CSP) headers for Swagger UI endpoints to allow script and style execution.
- Resolved Go toolchain version alignment and compilation compatibility in CI pipeline.

---

## [0.3.0] - 2026-07-13

### Added
- Enterprise observability suite with isolated Prometheus metrics registry and Grafana dashboard provisioning.
- Kubernetes-compatible health check probes (`/health/live`, `/health/ready`, `/health/startup`) with degraded state handling.
- Graceful server shutdown protocol draining worker pool channels and closing PostgreSQL/Redis pools safely.
- Security headers middleware enforcing CSP, HSTS, X-Frame-Options, and Referrer-Policy.
- Comprehensive Go benchmark test suite (`internal/benchmark/`) validating cryptographic, caching, and serialization performance.
- Standardized RFC 7807 problem details error responses and structured API versioning.

---

## [0.2.0] - 2026-07-12

### Added
- Redis cache-aside caching layer with singleflight deduplication to eliminate cache stampedes.
- Asynchronous worker pool for decoupled, non-blocking click tracking and analytics ingestion.
- Role-based access control (RBAC) and active session management with multi-device tracking and revocation.
- Token-bucket rate limiting and configurable request timeouts.
- Concurrency hardening with read-write mutex protection across mock repositories and audit loggers.

---

## [0.1.0] - 2026-07-11

### Added
- Clean Architecture layered system design (Handlers, Services, Repositories, Domain Models).
- JWT authentication engine featuring Refresh Token Rotation (RTR) and bcrypt password hashing.
- Base62 short code generation with collision handling and custom alias support.
- Core URL redirection resolution with sub-15ms response latency.
- PostgreSQL persistence engine powered by GORM with automated migration schema files.
- GitHub Actions CI pipeline running tests with race detection, static analysis, and code formatting checks.
- Containerized development orchestration via Docker Compose.
