---
name: Bug report
about: Report a reproducible bug or unexpected behavior
title: "[BUG] "
labels: bug
assignees: ''
---

## Bug Description
A clear and concise description of the bug.

## Steps to Reproduce
1. Send `POST /api/v1/...` with body `{ ... }`
2. Observe response ...
3. Expected: ...
4. Actual: ...

## Environment
- **OS**: (e.g., macOS 14, Ubuntu 22.04, Windows 11)
- **Go Version**: (e.g., 1.24.0)
- **Docker Version** (if applicable): 
- **LinkPulse Version / Commit**: 

## Request / Response (Redacted)
Paste any relevant HTTP request/response snippets here. **Do not include credentials or tokens.**

```
GET /r/abc123
Host: localhost:8080
```

## Expected Behavior
What you expected to happen.

## Actual Behavior
What actually happened. Include error messages and log output (sanitized).

## Additional Context
Any other context, screenshots, or configuration details.
