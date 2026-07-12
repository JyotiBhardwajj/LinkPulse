# Observability & Monitoring Layer

LinkPulse incorporates a production-grade observability and monitoring system built with Prometheus and Grafana, providing full visibility into HTTP services, internal cache stores, background worker queues, GORM database queries, and custom business event lifetimes.

---

## Architecture Overview

The observability layer is designed around an isolated registry system adhering strictly to Clean Architecture boundaries:

- **Isolated Registry**: Avoids the global Prometheus registry pattern. Instead, every metrics tracker instance maintains its own dedicated `prometheus.Registry` to prevent registration panics, allow independent setups per test, and isolate production state.
- **Thread-Safety & Singleton**: The production metrics registry is instantiated exactly once using a thread-safe `sync.Once` pattern via `GetProductionMetrics(...)`.
- **Low-Cardinality Labels**: To protect long-running Prometheus servers from memory exhaustion, no metric is registered with high-cardinality values (e.g., short codes, IP addresses, URLs, user IDs, or request IDs).

---

## Exposed Metric Collectors

### 1. HTTP Metrics
- `linkpulse_api_http_requests_total` (CounterVec):
  - **Labels**: `method`, `route` (matched Gin route templates via `c.FullPath()`), `status` (HTTP response status code).
  - **Cardinality protection**: If a route cannot be mapped (such as a 404), it falls back to `"unknown"`.
- `linkpulse_api_http_request_duration_seconds` (HistogramVec):
  - **Labels**: `method`, `route`.
  - Uses sensible production histogram buckets: `0.005s`, `0.01s`, `0.025s`, `0.05s`, `0.1s`, `0.25s`, `0.5s`, `1s`, `2.5s`, `5s`, `10s`.

### 2. Cache Metrics
- `linkpulse_api_redis_cache_hits_total` (Counter): Cumulative Redis cache lookup hits.
- `linkpulse_api_redis_cache_misses_total` (Counter): Cumulative Redis cache lookup misses.
- `linkpulse_api_redis_cache_errors_total` (Counter): Cumulative connection and deserialization failures.

### 3. Worker Pool Metrics
- `linkpulse_api_worker_events_processed_total` (Counter): Cumulative background events processed successfully.
- `linkpulse_api_worker_events_dropped_total` (Counter): Cumulative events dropped due to queue congestion.
- `linkpulse_api_worker_queue_size` (Gauge): Current count of elements in the buffered worker pool channel.
- `linkpulse_api_worker_active_workers` (Gauge): Number of active worker goroutines.

### 4. GORM Database Metrics
- `linkpulse_api_db_query_duration_seconds` (HistogramVec):
  - **Labels**: `repository` (the GORM database table/model repository), `operation` (`create`, `query`, `update`, `delete`, `row`).
  - **Centralized Plugin**: DB metrics are collected automatically using a custom GORM plugin callbacks registry (`Before`/`After` hooks) instead of polluting repository code.

### 5. Business Metrics
- `linkpulse_api_auth_login_success_total` (Counter): Cumulative successful user logins.
- `linkpulse_api_auth_login_failure_total` (Counter): Cumulative failed login attempts.
- `linkpulse_api_auth_refresh_success_total` (Counter): Cumulative successful token refreshes.
- `linkpulse_api_auth_refresh_failure_total` (Counter): Cumulative failed token rotations.
- `linkpulse_api_auth_logout_total` (Counter): Cumulative logouts.
- `linkpulse_api_links_created_total` (Counter): Cumulative shortened links created.
- `linkpulse_api_links_updated_total` (Counter): Cumulative links updated.
- `linkpulse_api_links_deleted_total` (Counter): Cumulative links deleted.
- `linkpulse_api_links_resolved_total` (Counter): Cumulative successful link redirections.
- `linkpulse_api_analytics_writes_total` (Counter): Cumulative successful background DB analytics writes.
- `linkpulse_api_analytics_errors_total` (Counter): Cumulative failures in background analytics processing.

---

## Local Configuration & Deployment

### 1. Configuration Keys
Configure the metrics layer in your `.env` file:
```ini
ENABLE_METRICS=true
METRICS_NAMESPACE=linkpulse
METRICS_SUBSYSTEM=api
```

### 2. Scraping Endpoint
Prometheus scrapes the API metrics route `/metrics` mounted directly on the root router.
- **Exposed Port**: `8080` (default)
- **Scrape Path**: `http://localhost:8080/metrics`
- **Bypass Rule**: The router automatically bypasses normal business logging, rate limiting, and request timeouts for `/metrics` requests to ensure zero side-effects.

### 3. Docker Compose Orchestration
Run the complete container stack using Docker Compose:
```bash
docker compose up -d
```
This boots up:
1. PostgreSQL Database (`linkpulse-db` on port `5432`)
2. Redis Cache (`linkpulse-redis` on port `6379`)
3. LinkPulse HTTP API (`linkpulse-api` on port `8080`)
4. Prometheus Scraper (`linkpulse-prometheus` on port `9090` using configuration [deploy/prometheus.yml](file:///c:/Users/jyoti/Desktop/LinkForge/deploy/prometheus.yml))
5. Grafana Dashboard (`linkpulse-grafana` on port `3000` provisioned with dashboard schema [deploy/grafana-dashboard.json](file:///c:/Users/jyoti/Desktop/LinkForge/deploy/grafana-dashboard.json))
