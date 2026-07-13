# LinkPulse API Pagination Specification

LinkPulse enforces standardized cursor/offset-based pagination metadata across all GET listing endpoints (e.g. `GET /api/v1/links`). This layout allows developers to build robust collection-traversing features.

---

## 1. Request Query Parameters

All listing API endpoints support the following sorting and pagination parameters:

| Parameter | Type | Default | Validation Bounds | Description |
| :--- | :--- | :--- | :--- | :--- |
| `page` | `integer` | `1` | `min=1` | The offset index block to return. |
| `limit` | `integer` | `20` | `min=1, max=100` | Number of items to return in a single page. |
| `search` | `string` | `""` | `max=255` | Case-insensitive matching string. |
| `sort` | `string` | `created_at` | `oneof=created_at updated_at expires_at` | Database field column mapping to sort on. |
| `order` | `string` | `desc` | `oneof=asc desc` | Direction sequence ordering. |

---

## 2. Response Metadata Schema

Every paginated response wraps results inside the standard JSON envelope containing a `metadata` object:

```json
{
  "success": true,
  "message": "Links retrieved successfully",
  "data": [
    {
      "id": "e98ca0b1-8b22-4911-aa99-c9a8db41e62a",
      "original_url": "https://google.com",
      "short_code": "ggl",
      "short_url": "http://localhost:8080/r/ggl"
    }
  ],
  "metadata": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5,
    "has_next": true,
    "has_previous": false
  },
  "request_id": "8dfa0f62-56a7-a8d3-b56a-688b9ffbdac0"
}
```

### Metadata Fields:

* `page`: Current page offset number.
* `page_size`: Active items limits count configured for this response.
* `total`: Combined count of matching resources across all pagination offsets.
* `total_pages`: Total page subsets available based on current page size.
* `has_next`: True if another page is available.
* `has_previous`: True if previous offset data page exists.
