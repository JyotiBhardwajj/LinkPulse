# LinkPulse Deployment Documentation

This document explains the steps to package, compile, and run the **LinkPulse** service.

---

## 1. Local Development Build

To run the application locally outside containers:
1. Ensure a PostgreSQL database and Redis instance are accessible on the host.
2. Initialize environment configurations:
   ```bash
   cp .env.example .env
   ```
3. Run migrations locally (requires `golang-migrate` command-line utility installed):
   ```bash
   migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/linkpulse_db?sslmode=disable" up
   ```
4. Run the Go server:
   ```bash
   go run ./cmd/api
   ```

---

## 2. Docker Compose Execution (Recommended)

Docker Compose orchestrates the compilation of the Go API container and links it to healthy Postgres and Redis images using named data volumes.

### Launch local stack
```bash
docker-compose up -d --build
```

This starts:
- **`linkpulse-db`** on port `5432` (with volume `postgres_data`)
- **`linkpulse-redis`** on port `6379` (with volume `redis_data`)
- **`linkpulse-api`** on port `8080` (waits for db and redis to be healthy before starting)

### Check Container Logs
```bash
docker-compose logs -f api
```

### Stop container stack
```bash
docker-compose down -v
```
*(Use `-v` to wipe data volumes if you want a clean reset of DB schema).*
