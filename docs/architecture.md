# LinkPulse Architecture Documentation

This document describes the architectural layout and design principles of **LinkPulse**, a production-grade URL Shortener and click analytics system written in Go.

---

## 1. Architectural Philosophy: Clean Architecture

LinkPulse follows **Clean Architecture** principles to isolate the core business logic from outer infrastructural dependencies (such as HTTP routing libraries, ORMs, and caching engines).

```
          ┌───────────────────────────────────────────────────┐
          │                  Infrastructure                   │
          │         (Gin Routing, GORM Database, Cache)       │
          └─────────────────────────┬─────────────────────────┘
                                    │
                                    ▼
          ┌───────────────────────────────────────────────────┐
          │                    Interfaces                     │
          │             (Handlers, Repositories)              │
          └─────────────────────────┬─────────────────────────┘
                                    │
                                    ▼
          ┌───────────────────────────────────────────────────┐
          │                 Application Core                  │
          │              (Service Layer Logic)                │
          └─────────────────────────┬─────────────────────────┘
                                    │
                                    ▼
          ┌───────────────────────────────────────────────────┐
          │                   Domain Entities                 │
          │                   (DB Models, DTOs)               │
          └───────────────────────────────────────────────────┘
```

### Dependency Rule
Dependencies flow **inward**. Outer packages (such as `routes` and `database`) import internal packages (such as `service` and `models`). Core business logic (`service`) remains independent of specific infrastructure, allowing simple unit testing by mocking repositories and cache layers.

---

## 2. Package Responsibilities

1. **`cmd/api/`**: The entrypoint module. Contains `main.go`, which simply instantiates the application coordinator.
2. **`internal/app/`**: Orchestrates dependency injection, configuration loads, database initialization, and graceful shutdown listeners.
3. **`internal/config/`**: Parses environmental configurations via Viper.
4. **`internal/database/`**: Manages GORM connection pool lifecycles.
5. **`internal/cache/`**: Manages Redis connectivity and provides the `LinkCache` abstraction.
6. **`internal/models/`**: Houses GORM structures and JSON transfer DTOs.
7. **`internal/repository/`**: Handles raw SQL query mapping. Exposes a `RepositoryManager` wrapper.
8. **`internal/service/`**: Houses core business requirements (hash codes, validation logic, resolution policies).
9. **`internal/handler/`**: Encapsulates request binding, JSON serialization, and response mapping.
10. **`internal/routes/`**: Pipelines request filters, logging middlewares, and maps HTTP verbs to controllers.
11. **`internal/utils/`**: Reusable generic helpers (Base62 generator, standard error responses, hashes).

---

## 3. High-Throughput Redirection Design

Redirection performance is a key scale criteria. To achieve sub-15ms resolution:
1. **Cache-First Check**: Handlers verify the existence of the mapping in Redis. If found, a `302 Found` redirection is returned immediately.
2. **Postgres Fallback**: Cache misses read from PostgreSQL and backfill Redis.
3. **Asynchronous Click Analytics**: Click events are dispatched inside a concurrent background goroutine. The client redirect is completed instantly and is not delayed by geolocation lookups or database writes.
