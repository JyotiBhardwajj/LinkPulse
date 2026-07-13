# LinkPulse API Examples

This guide provides examples of how to interact with the LinkPulse API using standard developers CLI tools (`curl` and `http` from HTTPie) and Postman request layout settings.

All requests require versioning routing under `/api/v1` (except system diagnostics `/health`, `/ready`, `/metrics`, and short code redirects `/r/:code`).

---

## 1. Authentication Endpoints

### Register Account (`POST /api/v1/auth/register`)

**curl**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"developer@example.com", "password":"supersecurepwd123"}'
```

**HTTPie**:
```bash
http POST http://localhost:8080/api/v1/auth/register \
  email=developer@example.com password=supersecurepwd123
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

---

### Login (`POST /api/v1/auth/login`)

**curl**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Device-Name: Developer Laptop" \
  -d '{"email":"developer@example.com", "password":"supersecurepwd123"}'
```

**HTTPie**:
```bash
http POST http://localhost:8080/api/v1/auth/login \
  X-Device-Name:"Developer Laptop" \
  email=developer@example.com password=supersecurepwd123
```

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "f47ac10b-...",
    "expires_in": 900
  },
  "request_id": "8dfa0f62-56a7-a8d3-b56a-688b9ffbdac0"
}
```

---

### Token Refresh (`POST /api/v1/auth/refresh`)

**curl**:
```bash
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"f47ac10b-..."}'
```

**HTTPie**:
```bash
http POST http://localhost:8080/api/v1/auth/refresh \
  refresh_token=f47ac10b-...
```

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Token refreshed successfully",
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "a82bc91b-...",
    "expires_in": 900
  },
  "request_id": "8dfa0f62-56a7-a8d3-b56a-688b9ffbdac0"
}
```

---

### Active Sessions (`GET /api/v1/auth/sessions`)

**curl**:
```bash
curl -X GET http://localhost:8080/api/v1/auth/sessions \
  -H "Authorization: Bearer <access_token>"
```

**HTTPie**:
```bash
http GET http://localhost:8080/api/v1/auth/sessions \
  "Authorization: Bearer <access_token>"
```

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Active sessions retrieved successfully",
  "data": [
    {
      "session_id": "a92bd12a-89a1-432d-88f1-c91823ab491a",
      "device": "Developer Laptop",
      "browser": "Chrome",
      "os": "Windows 11",
      "ip_hash": "2fca3a1c...",
      "last_used": "2026-07-13T11:47:00Z",
      "created_at": "2026-07-13T11:00:00Z",
      "current_session": true
    }
  ],
  "request_id": "8dfa0f62-56a7-a8d3-b56a-688b9ffbdac0"
}
```

---

## 2. Links Endpoints

### Shorten URL (`POST /api/v1/links`)

**curl**:
```bash
curl -X POST http://localhost:8080/api/v1/links \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{"original_url":"https://github.com/JyotiBhardwajj/LinkPulse", "custom_alias":"linkpulse-src"}'
```

**HTTPie**:
```bash
http POST http://localhost:8080/api/v1/links \
  "Authorization: Bearer <access_token>" \
  original_url=https://github.com/JyotiBhardwajj/LinkPulse custom_alias=linkpulse-src
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

---

## 3. System Health & Performance Endpoints

### Liveness Diagnostics (`GET /health`)

**curl**:
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

### Readiness Diagnostics (`GET /ready`)

**curl**:
```bash
curl -X GET http://localhost:8080/ready
```

**Response (200 OK)**:
```json
{
  "success": true,
  "message": "Ready check successful",
  "data": {
    "status": "ready",
    "database": "connected",
    "redis": "connected",
    "worker_pool": "running",
    "timestamp": "2026-07-13T11:47:00Z"
  },
  "request_id": "8dfa0f62-56a7-a8d3-b56a-688b9ffbdac0"
}
```
