# LinkPulse Production & Operations Guide

This document outlines the operations, health monitoring, lifecycle hooks, and container configurations for LinkPulse in staging and production environments.

---

## 1. Diagnostic Health Endpoints

LinkPulse exposes three independent HTTP probes under `/health/*` designed for Kubernetes and external load balancer orchestrations. All health endpoints bypass logging, rate limiting, and normal authentication controls to keep monitoring overhead negligible.

### Liveness Endpoint
- **Path**: `GET /health/live`
- **Purpose**: Checks only whether the application process is running and capable of handling HTTP requests. It **does not** depend on external systems like PostgreSQL, Redis, or Worker Pools.
- **Contract**:
  - Returns `HTTP 200 OK` while the process is alive.
  - Returns `HTTP 500 Internal Server Error` only if a fatal runtime error prevents server operations.

### Readiness Endpoint
- **Path**: `GET /health/ready`
- **Purpose**: Validates whether the application is fully bootstrapped and capable of handling traffic.
- **Critical vs Optional Dependency Policy**:
  - **Critical dependencies**: `postgres`, `worker_pool`, `config`. If any of these fail, the readiness state becomes unhealthy.
  - **Optional dependencies**: `redis`, `metrics`. If optional checks fail, the overall status response is updated to `"degraded"`, but the endpoint continues to return `HTTP 200 OK` (so the instance is not evicted from load balancers).
- **Contract**:
  - Returns `HTTP 200 OK` if the system is fully healthy or optionally degraded.
  - Returns `HTTP 503 Service Unavailable` if any critical dependency is down, or if the system's global readiness state is flagged to false (e.g. during graceful shutdown).

### Startup Endpoint
- **Path**: `GET /health/startup`
- **Purpose**: Asserts if the initial loading, database migration checks, and component bootstrapping have succeeded.
- **Contract**:
  - Returns `HTTP 200 OK` after startup completes.
  - Returns `HTTP 503 Service Unavailable` while startup is still in progress.

---

## 2. Graceful Shutdown & Lifecycle

When a `SIGINT` or `SIGTERM` signal is received:
1. **Toggle Readiness**: Immediately flags the readiness state to `false`, causing subsequent `/health/ready` checks to fail with `503`. This signals ingress controllers/load balancers to stop sending traffic.
2. **HTTP Drain**: Calls `http.Server.Shutdown(ctx)` with a configurable timeout (`SHUTDOWN_TIMEOUT`), allowing in-flight requests to complete without accepting new ones.
3. **Queue Drain**: The background worker pool stops accepting new tasks and drains remaining enqueued tasks, waiting for workers to complete persistence before exiting.
4. **Connections Cleanup**: Gracefully shuts down connection pools for Redis, PostgreSQL, and structured logging writers.

---

## 3. Kubernetes Recommendations

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: linkpulse-api
spec:
  template:
    spec:
      containers:
      - name: linkpulse
        image: linkpulse:latest
        startupProbe:
          httpGet:
            path: /health/startup
            port: 8080
          failureThreshold: 30
          periodSeconds: 1
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          periodSeconds: 5
```
