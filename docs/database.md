# LinkPulse Database Documentation

This document describes the relational database schema implemented in PostgreSQL for **LinkPulse**.

---

## 1. Schema Diagram

```
   ┌─────────────────┐
   │      users      │
   ├─────────────────┤
   │ id (PK)         │ ◄───┐
   │ email           │     │ (1-to-many nullable)
   │ password_hash   │     │
   │ timestamps      │     │
   └─────────────────┘     │
                           │
   ┌─────────────────┐     │
   │      links      │     │
   ├─────────────────┤     │
   │ id (PK)         │ ◄───┼───┐
   │ original_url    │     │   │
   │ short_code (UI) │     │   │
   │ title           │     │   │ (1-to-many cascade)
   │ user_id (FK) ───┘     │   │
   │ is_active       │         │
   │ expires_at      │         │
   │ timestamps      │         │
   └─────────────────┘         │
                               │
   ┌─────────────────┐         │
   │    analytics    │         │
   ├─────────────────┤         │
   │ id (PK)         │         │
   │ link_id (FK) ───┼─────────┘
   │ clicked_at      │
   │ ip_hash         │
   │ geo metadata    │
   │ user_agent      │
   └─────────────────┘
```

---

## 2. Table Schemas

### A. Table: `users`
Tracks system user accounts.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | Unique user identifier |
| `email` | VARCHAR(255) | UNIQUE, NOT NULL | User email address |
| `password_hash`| VARCHAR(255) | NOT NULL | Bcrypt password hash |
| `created_at` | TIMESTAMPTZ | NOT NULL | Records insertion time |
| `updated_at` | TIMESTAMPTZ | NOT NULL | Records last modification time |
| `deleted_at` | TIMESTAMPTZ | INDEXED | Supports GORM soft deletion |

---

### B. Table: `links`
Contains mapped links and settings.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | Unique link identifier |
| `original_url` | TEXT | NOT NULL | Destination URL |
| `short_code` | VARCHAR(50) | UNIQUE, INDEXED, NOT NULL | Unique route slug |
| `title` | VARCHAR(255) | NULL | Custom metadata title |
| `user_id` | UUID | FOREIGN KEY REFERENCES users(id) | Nullable link owner |
| `is_active` | BOOLEAN | DEFAULT TRUE, NOT NULL | Controls route availability |
| `expires_at` | TIMESTAMPTZ | NULL | Configured expiry time |
| `created_at` | TIMESTAMPTZ | NOT NULL | Records creation time |
| `updated_at` | TIMESTAMPTZ | NOT NULL | Records update time |
| `deleted_at` | TIMESTAMPTZ | INDEXED | Supports soft-delete |

---

### C. Table: `analytics`
Append-only log of redirect occurrences.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | PRIMARY KEY | Unique event identifier |
| `link_id` | UUID | INDEXED, FK REFERENCES links(id) ON DELETE CASCADE | Target link mapping |
| `clicked_at` | TIMESTAMPTZ | INDEXED, NOT NULL | Precise redirection time |
| `ip_hash` | VARCHAR(64) | NOT NULL | Anonymized IP hash |
| `country` | VARCHAR(100) | NULL | Geolocation country |
| `city` | VARCHAR(100) | NULL | Geolocation city |
| `browser` | VARCHAR(100) | NULL | Extracted browser type |
| `os` | VARCHAR(100) | NULL | Extracted operating system |
| `device` | VARCHAR(100) | NULL | Extracted device type |
| `referrer` | TEXT | NULL | Reference referrer header |
| `user_agent` | TEXT | NULL | User agent raw string |
