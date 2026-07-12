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
- **Performance Layer Design**: 
  - Resolves links utilizing a Redis Cache-Aside pattern (singleflight protected against concurrent database stampedes).
  - Fires click metrics tracking events asynchronously to a background Go worker pool channel buffer (non-blocking drop policy).
  
- **URL**: `/r/:code`
- **Method**: `GET`
- **Headers**: None
- **Response**: `302 Found` (Redirects to target location)
  - Expired links return `410 Gone`.
  - Deactivated or missing links return `404 Not Found`.

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

---

## 3. Analytics Operations

All analytics endpoints require authentication.

### Portfolio Analytics Overview
Returns aggregated statistics for all links belonging to the authenticated user.

- **URL**: `/api/v1/analytics/overview`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <JWT_TOKEN>`
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": {
      "total_links": 12,
      "active_links": 9,
      "inactive_links": 3,
      "total_clicks": 1420,
      "today_clicks": 45,
      "last_7_days_clicks": 320,
      "last_30_days_clicks": 1150
    }
  }
  ```

---

### Time-Series Click History
Returns zero-filled click history timelines.

- **URL**: `/api/v1/analytics/clicks`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <JWT_TOKEN>`
- **Query Parameters**:
  - `start_date`: RFC3339 format start timestamp (optional, default 30d ago)
  - `end_date`: RFC3339 format end timestamp (optional, default now)
  - `interval`: `hour`, `day`, `week`, `month` (optional, default `day`)
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": [
      { "timestamp": "2026-07-11", "clicks": 15 },
      { "timestamp": "2026-07-12", "clicks": 0 },
      { "timestamp": "2026-07-13", "clicks": 7 }
    ]
  }
  ```

---

### Top Performing Links
Returns shortened links ordered by total click counts.

- **URL**: `/api/v1/analytics/top-links`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <JWT_TOKEN>`
- **Query Parameters**:
  - `limit`: Clamped integer (default 10, max 100)
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": [
      {
        "short_code": "goog",
        "original_url": "https://google.com",
        "click_count": 890,
        "last_clicked_at": "2026-07-12T14:40:00Z"
      }
    ]
  }
  ```

---

### Device Distribution
Returns device platform percentages and counts.

- **URL**: `/api/v1/analytics/devices`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <JWT_TOKEN>`
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": [
      { "name": "Desktop", "count": 60, "percentage": 60.0 },
      { "name": "Mobile", "count": 30, "percentage": 30.0 },
      { "name": "Tablet", "count": 10, "percentage": 10.0 },
      { "name": "Unknown", "count": 0, "percentage": 0.0 }
    ]
  }
  ```

---

### Detailed Single Link Analytics
Returns full metrics suite for a specific shortened link ID.

- **URL**: `/api/v1/links/:id/analytics`
- **Method**: `GET`
- **Headers**: `Authorization: Bearer <JWT_TOKEN>`
- **Response**: `200 OK`
  ```json
  {
    "success": true,
    "data": {
      "link_id": "e2a3b190-67c4-4b53-a55e-a2f01a357bb1",
      "original_url": "https://google.com",
      "short_code": "goog",
      "total_clicks": 100,
      "clicks_over_time": [
        { "timestamp": "2026-07-12", "clicks": 100 }
      ],
      "browser_distribution": [
        { "name": "Chrome", "count": 80, "percentage": 80.0 },
        { "name": "Firefox", "count": 20, "percentage": 20.0 }
      ],
      "device_distribution": [
        { "name": "Desktop", "count": 100, "percentage": 100.0 }
      ],
      "top_referrers": [
        { "name": "Direct/Unknown", "count": 100, "percentage": 100.0 }
      ]
    }
  }
  ```

