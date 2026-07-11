# LinkPulse API Documentation

This document outlines the API specifications for the LinkPulse service.

---

## 1. System Diagnostics

### Health Check
Returns database and cache statuses and latency metrics.

- **URL**: `/health`
- **Method**: `GET`
- **Headers**: None
- **Response**: `200 OK`
  ```json
  {
    "status": "healthy",
    "postgres": {
      "status": "up",
      "latency_ms": 4
    },
    "redis": {
      "status": "up",
      "latency_ms": 1
    },
    "version": "1.0.0",
    "uptime_seconds": 120,
    "timestamp": "2026-07-11T12:00:00Z"
  }
  ```

---

## 2. Link Operations

### Shorten Link
Creates a shortened link code mapping to the original target.

- **URL**: `/api/v1/links`
- **Method**: `POST`
- **Headers**: `Content-Type: application/json`
- **Request Body**:
  ```json
  {
    "original_url": "https://www.google.com",
    "title": "Google Search",
    "expires_at": "2026-12-31T23:59:59Z",
    "custom_slug": "goog"
  }
  ```
- **Response**: `201 Created`
  ```json
  {
    "success": true,
    "data": {
      "id": "e2a3b190-67c4-4b53-a55e-a2f01a357bb1",
      "original_url": "https://www.google.com",
      "short_code": "goog",
      "short_url": "http://localhost:8080/r/goog",
      "title": "Google Search",
      "expires_at": "2026-12-31T23:59:59Z",
      "created_at": "2026-07-11T11:42:08Z"
    }
  }
  ```

---

### Redirect Link
Resolves a shortened link code and redirects to the original URL.

- **URL**: `/r/:code`
- **Method**: `GET`
- **Headers**: None
- **Response**: `302 Found` (Redirects to target location)

---

### Link Statistics
Fetches redirection metrics.

- **URL**: `/api/v1/links/:code/stats`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <JWT_TOKEN>`
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": {
      "id": "e2a3b190-67c4-4b53-a55e-a2f01a357bb1",
      "short_code": "goog",
      "original_url": "https://www.google.com",
      "total_clicks": 142
    }
  }
  ```
