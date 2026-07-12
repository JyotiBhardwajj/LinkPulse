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

### E. Background Cleanup Scheduler Ticker Flow
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

### Authentication & Session Management

#### Register a New User
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepassword123"}'
```

#### User Login (Obtain Tokens)
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "securepassword123"}'
```
**Response (`200 OK`)**:
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOiJIUzI1Ni...",
    "refresh_token": "9a38fcd8e...",
    "expires_in": 900
  }
}
```

#### Rotate Session (Refresh Token Rotation)
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "9a38fcd8e..."}'
```

#### User Logout
```bash
curl -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "9a38fcd8e..."}'
```

---

### Link Management (Authenticated)

#### Create a Shortened Link
```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://news.ycombinator.com",
    "title": "Hacker News",
    "custom_alias": "hn-home",
    "expires_at": "2026-12-31T23:59:59Z"
  }'
```
**Response (`201 Created`)**:
```json
{
  "success": true,
  "message": "Link shortened successfully",
  "data": {
    "id": "2db42907-9b2f-4886-bb21-2e6fb082161f",
    "original_url": "https://news.ycombinator.com",
    "short_code": "hn-home",
    "short_url": "http://localhost:8080/r/hn-home",
    "title": "Hacker News",
    "expires_at": "2026-12-31T23:59:59Z",
    "is_active": true,
    "click_count": 0,
    "created_at": "2026-07-11T13:31:00Z",
    "updated_at": "2026-07-11T13:31:00Z"
  }
}
```

#### List My Links (Paginated, Search & Sorted)
- **Query parameters**:
  - `page` (default: 1)
  - `limit` (default: 20, max: 100)
  - `search` (optional search keyword)
  - `sort` (whitelist: `created_at`, `updated_at`, `expires_at`)
  - `order` (`asc`, `desc`)
  - `status` (`active`, `expired`, `inactive`, `deleted`)
```bash
curl -X GET "http://localhost:8080/api/v1/links?page=1&limit=2&search=Hacker&sort=created_at&order=desc" \
  -H "Authorization: Bearer <access_token>"
```
**Response (`200 OK`)**:
```json
{
  "success": true,
  "message": "Links retrieved successfully",
  "data": {
    "page": 1,
    "limit": 2,
    "total": 1,
    "total_pages": 1,
    "items": [
      {
        "id": "2db42907-9b2f-4886-bb21-2e6fb082161f",
        "original_url": "https://news.ycombinator.com",
        "short_code": "hn-home",
        "short_url": "http://localhost:8080/r/hn-home",
        "title": "Hacker News",
        "expires_at": "2026-12-31T23:59:59Z",
        "is_active": true,
        "click_count": 0,
        "created_at": "2026-07-11T13:31:00Z",
        "updated_at": "2026-07-11T13:31:00Z"
      }
    ]
  }
}
```

#### Get Link Details
```bash
curl -X GET http://localhost:8080/api/v1/links/2db42907-9b2f-4886-bb21-2e6fb082161f \
  -H "Authorization: Bearer <access_token>"
```

#### Update Link (PATCH)
```bash
curl -X PATCH http://localhost:8080/api/v1/links/2db42907-9b2f-4886-bb21-2e6fb082161f \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "Updated Hacker News Title", "is_active": false}'
```

#### Delete Link (Soft Delete)
```bash
curl -X DELETE http://localhost:8080/api/v1/links/2db42907-9b2f-4886-bb21-2e6fb082161f \
  -H "Authorization: Bearer <access_token>"
```
**Response (`204 No Content`)**

---

### Redirection (Public)

#### Resolve Shortened URL
```bash
curl -i -X GET http://localhost:8080/r/hn-home
```
**Response (`302 Found` with Location redirection header)**:
```text
HTTP/1.1 302 Found
Location: https://news.ycombinator.com
```
*Edge cases*:
- Expired links return `410 Gone`.
- Deactivated or soft-deleted links return `404 Not Found`.

---

## 5. Performance Optimizations

1. **Write Contention Prevention & Go Worker Pool**: Click events are sent to a bounded channel and processed by a pool of background worker goroutines. If the queue channel fills up, events are dropped with a warning log, ensuring zero redirect latency impact.
2. **Read Path Redirection Speed**: Redirection requests read from Redis cache, completing in <1ms without touching PostgreSQL.
3. **Cache Stampede Prevention (singleflight)**: Parallel cache misses on the same key are merged using Go's `singleflight.Group` so only one SQL query is run against Postgres while concurrent waiters share the resulting cache write.
4. **Smarter Cache TTL**: The cache write TTL is dynamically calculated as `min(CACHE_TTL, remaining lifetime)` to ensure expired records do not persist in Redis memory.
5. **Optimized Indexes**:
   - Composite index `(link_id, clicked_at)` speeds up aggregation queries tracking clicks within date bounds.
   - B-tree indices on `short_code` and `user_id` allow O(1) matching during redirects and user lists.
   - Composite index `(is_active, expires_at)` supports efficient background deactivations of expired records without scanning the entire table.
6. **SkipDefaultTransaction**: GORM defaults to running every single write inside a transaction. We disable this (`SkipDefaultTransaction: true`) for performance since our repositories manage transactions manually.

---

## 6. Security Decisions

1. **Client IP Anonymization**: Storing client IP addresses violates GDPR. We hash IP addresses using a secure SHA-256 algorithm (`ip_hash`) before saving, satisfying compliance while allowing user-agent unique metrics.
2. **Container Runtime Sandbox**: The Go API runtime container runs as a non-privileged `appuser`, preventing privilege escalations in case of remote code execution (RCE) bugs.
3. **Structured Context Traceability**: The Request ID middleware assigns a unique UUID to every thread. The `slog` logger middleware includes this trace ID in all output rows, allowing logs to be filtered easily in ELK/Grafana stacks.

---

## 7. Trade-offs & Engineering Decisions

- **Viper vs Env Variables**: Viper allows reading settings from files locally while letting production systems override them via environment variables seamlessly.
- **Gin vs Standard `net/http`**: Gin is used for its highly optimized rad-tree routing engine and JSON binding conveniences. While standard library `net/http` is cleaner, Gin's performance on routing match parameters is production-tested.
- **Bounded Worker Pool Queue vs Redis Streams/Apache Kafka**: We implemented click tracking asynchronously using a context-aware worker pool running on a bounded in-memory Go channel. While this prevents blocking redirection threads, it has limited buffer memory capacity. Future scaling stages will replace this in-memory channel with a distributed log like **Redis Streams** or **Apache Kafka** to support durability, replication, and scaling across multiple API nodes without code refactor.

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

---

## 9. Analytics Engine Architecture & Optimizations

### A. Analytics Request Flow
```
[Client Request]
       │
       ▼
[Handler: Date Validation & Limit Clamping]
       │
       ▼
[Service: Timezone Realignment & Zero-Fill Logic]
       │
       ▼
[Repository: Raw SQL Aggregation Queries]
       │
       ▼
[PostgreSQL Engine (targeting indices: idx_analytics_link_clicked)]
```

### B. Aggregation Strategy & GORM Bypass Rationale
LinkPulse bypasses GORM's built-in query generation and uses **Raw SQL** for analytics aggregation for three reasons:
1. **Time-Series Truncations**: Grouping by variable intervals (hour, day, week, month) requires database-level time truncations like `date_trunc` which are specific to PostgreSQL and hard to compile cleanly using standard ORM query builders.
2. **Referring Domain Extraction**: Aggregating referrers by domain rather than complete query URLs utilizes regular expression matching: `regexp_replace(referrer, '^https?://([^/]+).*$', '\1')`.
3. **Optimized execution**: Explicit Raw SQL prevents GORM from adding default wrappers, ensuring PostgreSQL uses index scans (`idx_analytics_link_clicked`) instead of expensive sequential scans.

### C. Expected Query Complexity
- **Overview Stats**: $O(K)$ where $K$ is the number of links matching `user_id`. PostgreSQL processes count queries using Index-Only scans.
- **Timeline/Distributions**: $O(\log N + M)$ where $N$ is total click events in `analytics` and $M$ is the number of matching records in the requested date range. Scoped via index `(link_id, clicked_at)`.

### D. Future Analytics Scalability Roadmap
1. **Materialized Views**: Compute browser/device ratios periodically (e.g. hourly) to offload real-time aggregation queries.
2. **Click Aggregation Tables**: Store pre-aggregated daily/hourly counters for shortened links to avoid querying raw click logs for dashboards.
3. **PostgreSQL Partitioning**: Partition the `analytics` table horizontally by month/year bounds to keep indices small and facilitate fast archival drops.
4. **Redis Leaderboards**: Cache top links in sorted sets (`ZSET`) updated in real-time by the worker pool to bypass PostgreSQL entirely for top-links queries.

