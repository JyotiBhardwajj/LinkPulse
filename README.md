# LinkPulse: Production-Grade Link Management Platform

LinkPulse is a high-performance, enterprise-ready URL redirection and analytics platform built in Go. Adhering strictly to **Clean Architecture** principles, the service is optimized for low latency (<15ms redirects), data privacy, and horizontal scalability.

This repository serves as a showcase for production-grade backend engineering practices, making it an excellent resource for backend systems design discussions and SDE portfolio reviews.

---

## 1. System Architecture & Diagrams

### A. Architecture Diagram
LinkPulse organizes code dependencies into clean concentric circles where control flows outward to inward, maintaining complete mockability of business domains:

```
+-------------------------------------------------------------------------+
|                              INFRASTRUCTURE                             |
|        Gin Web Server | GORM Postgres Engine | Redis Cache Client       |
|                                                                         |
|         +-----------------------------------------------------+         |
|         |                     INTERFACES                      |         |
|         |        HTTP Controllers | GORM SQL Repositories     |         |
|         |                                                     |         |
|         |         +---------------------------------+         |         |
|         |         |         APPLICATION CORE        |         |         |
|         |         |      Domain Service Interfaces  |         |         |
|         |         |                                 |         |         |
|         |         |         +-------------+         |         |         |
|         |         |         |   DOMAIN    |         |         |         |
|         |         |         | DB Entities |         |         |         |
|         |         |         +-------------+         |         |         |
|         |         +---------------------------------+         |         |
|         +-----------------------------------------------------+         |
+-------------------------------------------------------------------------+
```

---

### B. Request Sequence Diagram (Redirection Path)
The sequence below illustrates the execution flow for link resolving. Note how redirect logic executes instantly, offloading analytics asynchronously to prevent transaction delays:

```
[Client]             [API Engine]          [Redis Cache]       [Postgres DB]
   │                      │                      │                   │
   │─── GET /r/:code ────►│                      │                   │
   │                      │───── cache.Get() ───►│                   │
   │                      │◄──── Cache Hit ──────│                   │
   │                      │                      │                   │
   │                      │ (Cache Miss)         │                   │
   │                      │─────────────────────────────────────────►│
   │                      │◄─────────── db.FindByCode() ─────────────│
   │                      │                      │                   │
   │                      │───── cache.Set() ───►│                   │
   │                      │                      │                   │
   │◄── 302 Redirect ─────│                      │                   │
   │                      │                      │                   │
   │ (Background Thread)  │                      │                   │
   │                      │───── RecordClick() ─────────────────────►│
   │                      │◄──── Insert Success ─────────────────────│
```

---

### C. Caching Flow Diagram (Cache-Aside with Singleflight)
Our caching model ensures database read load is minimized and protects against cache stampedes during viral concurrent hit events:

```
                       [Incoming Request]
                               │
                               ▼
                    [Check Redis Cache key]
                     /                 \
             (Hit)  /                   \  (Miss)
                   v                     v
            [Return URL]      [singleflight.Do(code)]
                               /                 \
                     (Shared Get)               (Query DB)
                          /                          │
                         v                           ▼
                    [Return URL]               [Write Cache]
                                                     │
                                                     ▼
                                                [Return URL]
```

---

### D. Worker Pool Flow Diagram
Redirections offload analytics asynchronously to prevent transaction blocking:

```
[Incoming Request] ──► [HTTP Redirect]
                              │
                      (Async Submit)
                              │
                              ▼
                [Bounded queue channel] ──► (Queue full?) ──► Yes ──► [Drop Event & slog.Warn]
                              │
                          No (Accept)
                              │
                              ▼
                   [Worker pool workers] ──► [DB Insert Analytics]
```

---

### E. Background Cleanup Ticker Flow
Expired links are periodically identified and cache keys are invalidated:

```
[time.NewTicker] ──► [Query Active Expired links]
                                │
                       (Composite Index query)
                                │
                                ▼
                   [Batch DB is_active=false]
                                │
                                ▼
                     [Cache Key Deletes]
                                │
                                ▼
                        [slog.Info Summary]
```

---

### F. Database ER Diagram

```
  users (Primary Account Data)
  +--------------------------+
  | PK  id            UUID   | 
  | UQ  email         VARCHAR| <---+
  |     password_hash VARCHAR|     |
  |     timestamps           |     | (1-to-many, nullable)
  +--------------------------+     |
                                   |
  links (Link Configurations)      |
  +--------------------------+     |
  | PK  id            UUID   |     |
  |     original_url  TEXT   |     |
  | UQ  short_code    VARCHAR|     |
  |     title         VARCHAR|     |
  | FK  user_id       UUID   |-----+
  |     is_active     BOOLEAN| <---+
  |     expires_at    TZ     |     |
  |     timestamps           |     | (1-to-many, cascade delete)
  +--------------------------+     |
                                   |
  analytics (Click Tracking Logs)  |
  +--------------------------+     |
  | PK  id            UUID   |     |
  | FK  link_id       UUID   |-----+
  |     clicked_at    TZ     |
  |     ip_hash       VARCHAR|
  |     geo_metadata  VARCHAR|
  |     user_agent    TEXT   |
  +--------------------------+
```

---

## 2. API Versioning & Architecture

LinkPulse supports route versioning under `/api/v1` to decouple evolving routes from business handlers.
- Router is split into distinct registration functions: `registerV1Routes` and `registerV2Routes`.
- Future api expansions (e.g. `/api/v2`) can be mounted alongside existing groups without modification of underlying V1 services or handlers.
- Handlers remain entirely version-agnostic.

---

## 3. Developer Quick Start

### Local Prerequisites
- Go 1.25.x
- Docker and Docker Compose

### Run Local Stack
1. Clone the repository and navigate to the project directory:
   ```bash
   cd LinkForge
   ```
2. Build and start backing containers (Postgres, Redis, Prometheus, Grafana) along with the API engine:
   ```bash
   docker-compose up -d --build
   ```
3. Run migrations locally if running without Docker:
   ```bash
   make migrate-up
   ```

---

## 4. Authentication Flow

Authentication follows the standard OAuth2 Bearer pattern with JWT Token Rotation:
1. **Register**: Send a email and password payload to `POST /api/v1/auth/register` to create a user account.
2. **Login**: Authenticate at `POST /api/v1/auth/login` to obtain an `access_token` and `refresh_token`. Include the client name in the `X-Device-Name` header.
3. **Authorized Requests**: Include the `Authorization: Bearer <access_token>` header on all protected endpoints.
4. **Token Rotation**: Send the `refresh_token` to `POST /api/v1/auth/refresh` before the access token expires (900 seconds) to receive a fresh token pair.

---

## 5. OpenAPI Specs & SDK Generation

LinkPulse provides a production-grade OpenAPI 3.0.3 specification, featuring descriptive unique `operationId` parameters, error model schemas, headers documentation, and example bodies.

### Download Specification
Retrieve the OpenAPI document directly from a running instance:
```bash
curl -o swagger.json http://localhost:8080/docs/swagger.json
```

### SDK Code Generation
You can compile native SDK packages for TypeScript, Java, Python, Go, and Rust using **OpenAPI Generator** or **Swagger Codegen**.

For example, to generate a TypeScript Axios client:
```bash
docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
    -i /local/docs/swagger.json \
    -g typescript-axios \
    -o /local/sdk/typescript
```

---

## 6. Observability & Performance Metrics

Observability indicators are exposed natively for Prometheus scraping at `GET /metrics`. This endpoint runs outside the normal business middleware stack, preventing scraping load from being logged or rate limited.

### Configured Metrics:
* **HTTP Performance**: Request count by code and route, latency histograms.
* **Database Queries**: Latency histograms grouped by SQL table and operation (select, query, update, delete).
* **Caching Performance**: Redis cache hits, misses, and read errors counters.
* **Worker Queue Pool**: In-flight queue size, active workers counters, processed/dropped click counts.

---

## 7. API Endpoints Examples

### System Health (`GET /health`)
```bash
curl -X GET http://localhost:8080/health
```
**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Health check successful",
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "git_commit": "8dfa0f6",
    "timestamp": "2026-07-13T11:47:00Z"
  },
  "request_id": "8dfa0f62-56a7-a8d3-b56a-688b9ffbdac0"
}
```

### Register User Account (`POST /api/v1/auth/register`)
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"developer@example.com", "password":"supersecurepwd123"}'
```
**Response (201 Created)**:
```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "id": "c9a64e42-7a42-4911-aa11-88f6356a5c1e",
    "email": "developer@example.com",
    "created_at": "2026-07-13T11:47:00Z"
  },
  "request_id": "8dfa0f62-56a7-a8d3-b56a-688b9ffbdac0"
}
```

### Shorten a Link (`POST /api/v1/links`)
```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"original_url":"https://github.com/JyotiBhardwajj/LinkPulse", "custom_alias":"linkpulse-src"}'
```
**Response (201 Created)**:
```json
{
  "success": true,
  "message": "Link shortened successfully",
  "data": {
    "id": "e98ca0b1-8b22-4911-aa99-c9a8db41e62a",
    "original_url": "https://github.com/JyotiBhardwajj/LinkPulse",
    "short_code": "linkpulse-src",
    "short_url": "http://localhost:8080/r/linkpulse-src",
    "is_active": true,
    "click_count": 0,
    "created_at": "2026-07-13T11:47:00Z",
    "updated_at": "2026-07-13T11:47:00Z"
  },
  "request_id": "8dfa0f62-56a7-a8d3-b56a-688b9ffbdac0"
}
```

### Validation Error Example (`422 Unprocessable Entity`)
```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"original_url":"invalid_url"}'
```
**Response (422 Unprocessable Entity - RFC7807 Format)**:
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Validation failed"
  },
  "type": "https://linkpulse.com/errors/validation-error",
  "title": "Validation Error",
  "status": 422,
  "detail": "One or more fields failed validation constraints.",
  "instance": "/api/v1/links",
  "details": [
    {
      "field": "original_url",
      "rule": "url",
      "message": "The original_url field must be a valid absolute URL (http or https)"
    }
  ],
  "request_id": "8dfa0f62-56a7-a8d3-b56a-688b9ffbdac0"
}
```

---

## 8. Directory Layout

- **`cmd/`**: houses main entrypoint definitions.
- **`deploy/`**: configuration mappings for monitoring engines (Prometheus / Grafana).
- **`docs/`**: API developer specification manuals (errors, pagination, examples, OpenAPI JSON, production).
- **`internal/app/`**: application initialization and dependency wiring.
- **`internal/config/`**: application environment structure setups.
- **`internal/database/`**: GORM driver initializations and plugins.
- **`internal/health/`**: parallel checker definitions and readiness states.
- **`internal/models/`**: schemas entities and structures properties.
- **`internal/repository/`**: persistence logic.
- **`internal/service/`**: core business rules logic.
- **`internal/handler/`**: payload binders and output formatters.
- **`internal/middleware/`**: routing filters chain.
- **`internal/utils/`**: helper functionalities (base62, hashing, response builders, validator realignments, ETags).

---

## 9. Production Operational Design

### Diagnostic Probes
LinkPulse implements three independent diagnostics routes under `/health/*` designed for container probe orchestrations.

1. **Liveness Probe** (`/health/live`): Check process execution. Returns `200` under active operations.
2. **Readiness Probe** (`/health/ready`): Check operational state of critical systems (`postgres`, `worker_pool`, `config`). Optional failures (`redis`, `metrics`) degrade the status response but continue to return `HTTP 200` to prevent unnecessary traffic interruption.
3. **Startup Probe** (`/health/startup`): Check bootstrapping. Returns `200` after migrations are verified and services have initialized.

### Graceful Shutdown
Toggling termination flags instantly flags `/health/ready` to `503`, allowing ingresses to route traffic elsewhere before existing requests are drained, background worker channels are completely processed, and SQL connection pools are safely closed.

---

## 10. Performance Testing & Load Validation

LinkPulse includes a production-grade load testing suite powered by **Grafana k6** to validate system performance, latency distributions, and concurrency stability.

### Available Load Scenarios

All load testing scripts reside in the `/loadtest` directory:
- **Smoke Test (`smoke.js`)**: Executes with a single Virtual User (VU) to verify basic API correctness and end-to-end functionality under zero load.
- **Baseline Test (`baseline.js`)**: Simulates typical production load patterns over a short duration to determine baseline performance metrics.
- **Stress Test (`stress.js`)**: Ramps up to elevated concurrent VU levels to pinpoint bottlenecks, evaluate connection pool limits, and locate resource limits.
- **Spike Test (`spike.js`)**: Floods the application with a high volume of traffic within seconds to verify system resilience and rapid autoscaling recovery.
- **Soak Test (`soak.js`)**: Sustains continuous load over an extended period to identify memory leaks, database connection leakage, or log accumulation issues.

### Running the Load Tests

Verify that your application is running locally (e.g. `make run` or inside Docker), then trigger k6 tests via the following `make` recipes:

```bash
# Run smoke validation
make load-smoke

# Run baseline profile
make load-baseline

# Run stress verification
make load-stress

# Run spike resilience test
make load-spike

# Run soak memory test
make load-soak
```

### Configuration & Environment Variables

You can configure the target server URL using the `BASE_URL` environment variable:
```bash
BASE_URL=https://linkpulse-production.up.railway.app k6 run loadtest/smoke.js
```

### Interpreting Reports & Thresholds

Each load test verifies the following performance parameters under strict **SLA Thresholds**:
1. **HTTP Error Rate**: Under 1.0% (`rate < 0.01`).
2. **HTTP Latency**: 95% of requests must complete under 500ms (`p(95) < 500`), and 99% under 1000ms (`p(99) < 1000`).

Upon execution, the test suite generates two artifacts in the `loadtest/` directory:
- **`loadtest/summary.json`**: Machine-readable JSON summary of metrics, throughput, latency distributions, and SLA check status.
- **`loadtest/report.html`**: A clean, responsive HTML report showcasing latency, total requests, throughput (RPS), and error percentages.


