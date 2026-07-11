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

### C. Caching Flow Diagram
Our caching model ensures database read load is minimized during high-traffic viral link events:

```
                       [Incoming Request]
                               │
                               ▼
                    [Check Redis Cache]
                     /             \
             (Hit)  /               \  (Miss)
                   v                 v
            [Return URL]      [Query PostgreSQL]
                               /             \
                     (Found)  /               \  (Expired/Not Found)
                             v                 v
                       [Write Redis]      [Return 404]
                             │
                             ▼
                        [Return URL]
```

---

### D. Database ER Diagram

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

## 2. Folder Explanation

- **`cmd/api/main.go`**: Program entry point. Houses the ~10 line initialization routine invoking `app.NewApplication()`.
- **`internal/app/`**: Orchestrates application bootstrapping, dependency injection, and SIGINT/SIGTERM listener handlers for graceful shutdown.
- **`internal/config/`**: Nested configuration structs parsed using Viper, supporting environment variable binding.
- **`internal/database/`**: Initializes GORM PostgreSQL connection pools (`SetMaxOpenConns`, `SetMaxIdleConns`).
- **`internal/cache/`**: Wraps the `go-redis` client and implements `LinkCache` interfaces.
- **`internal/models/`**: Houses GORM database entities and JSON verification DTOs.
- **`internal/repository/`**: Performs SQL mapping logic. Exposes a unified `RepositoryManager` store.
- **`internal/service/`**: Implements core business rules (Base62 generations, collision-retry loop, validation).
- **`internal/handler/`**: Inspects payload validations and formats API outputs.
- **`internal/routes/`**: Binds routes and sets up the middleware chain.
- **`internal/utils/`**: Shared helpers (standard envelope responses, base62 encoders, password hashes).

---

## 3. Docker Container Architecture

LinkPulse utilizes Docker Compose to define a network-isolated local testing stack:
- **`Go API`**: Custom multi-stage builder container compiled with static linking. Switches to a non-privileged `appuser` before startup.
- **`db`**: PostgreSQL 16 container utilizing healthcheck pings to verify database readiness before exposing ports.
- **`redis`**: Redis 7 container storing cache keys transiently.

---

## 4. API Endpoints & Examples

### System Health
```bash
curl -X GET http://localhost:8080/health
```
**Response (`200 OK`)**:
```json
{
  "success": true,
  "message": "Health check completed",
  "data": {
    "status": "healthy",
    "postgres": {
      "status": "up",
      "latency_ms": 3
    },
    "redis": {
      "status": "up",
      "latency_ms": 1
    },
    "version": "1.0.0",
    "git_commit": "3b29c9b",
    "environment": "development",
    "uptime_seconds": 360,
    "timestamp": "2026-07-11T13:30:00Z"
  }
}
```

### Shorten a URL
```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://news.ycombinator.com", "title": "Hacker News"}'
```
**Response (`201 Created`)**:
```json
{
  "success": true,
  "message": "Link shortened successfully",
  "data": {
    "id": "2db42907-9b2f-4886-bb21-2e6fb082161f",
    "original_url": "https://news.ycombinator.com",
    "short_code": "xR82d1K",
    "short_url": "http://localhost:8080/r/xR82d1K",
    "title": "Hacker News",
    "created_at": "2026-07-11T13:31:00Z"
  }
}
```

---

## 5. Performance Optimizations

1. **Write Contention Prevention**: NAIVE shorten architectures increment an integer counter (`clicks = clicks + 1`) directly in the link table. This requires row-locking updates. In LinkPulse, we log click events as append-only records in a separate `analytics` table, which converts row-locks into non-blocking inserts.
2. **Read Path Redirection Speed**: Redirection requests read from Redis cache, completing in <1ms without touching PostgreSQL.
3. **Optimized Indexes**:
   - Composite index `(link_id, clicked_at)` speeds up aggregation queries tracking clicks within date bounds.
   - B-tree indices on `short_code` and `user_id` allow O(1) matching during redirects and user lists.
4. **SkipDefaultTransaction**: GORM defaults to running every single write inside a transaction. We disable this (`SkipDefaultTransaction: true`) for performance since our repositories manage transactions manually.

---

## 6. Security Decisions

1. **Client IP Anonymization**: Storing client IP addresses violates GDPR. We hash IP addresses using a secure SHA-256 algorithm (`ip_hash`) before saving, satisfying compliance while allowing user-agent unique metrics.
2. **Container Runtime Sandbox**: The Go API runtime container runs as a non-privileged `appuser`, preventing privilege escalations in case of remote code execution (RCE) bugs.
3. **Structured Context Traceability**: The Request ID middleware assigns a unique UUID to every thread. The `slog` logger middleware includes this trace ID in all output rows, allowing logs to be filtered easily in ELK/Grafana stacks.

---

## 7. Trade-offs & Engineering Decisions

- **Viper vs Env Variables**: Viper allows reading settings from files locally while letting production systems override them via environment variables seamlessly.
- **Gin vs Standard `net/http`**: Gin is used for its highly optimized rad-tree routing engine and JSON binding conveniences. While standard library `net/http` is cleaner, Gin's performance on routing match parameters is production-tested.
- **Synchronous Goroutines vs Event Message Broker (Kafka/RabbitMQ)**: Click analytics are currently recorded in a background goroutine. Although an external message broker handles high-volume loads better, it introduces significant setup overhead. We designed `RecordClick` as a separate, isolated service method to ensure that switching to an asynchronous task manager (e.g. Celery-style Redis queue) requires zero refactoring.

---

## 8. Interview Questions & System Design Topics

Prepare these answers for SDE interviews when demonstrating this project:

1. **How does GORM handle soft deletes?**
   * *Answer*: By embedding `gorm.DeletedAt` in the model, GORM automatically appends `WHERE deleted_at IS NULL` to all queries, hiding them. Hard deletions can still be run using `Unscoped()`.
2. **Why separate the DB Models from Request DTOs?**
   * *Answer*: Placing validation rules (`binding:"required"`) directly on GORM models violates Separation of Concerns. It leaks API layer validations into the persistence layer. DTOs insulate the database structures from incoming requests.
3. **Why do we need a Repository Manager?**
   * *Answer*: It isolates database wire-up details in the bootstrap layer (`internal/app`). Services receive only the concrete interfaces they require, maintaining mockability without knowing how repos are constructed.
4. **How would you scale this URL shortener to handle 100k redirect requests per second?**
   * *Answer*: Increase Redis replicas to handle read distribution, use a consistent hashing load balancer to route requests, and offload background click writes by pushing analytics events into a distributed message log like Apache Kafka.
