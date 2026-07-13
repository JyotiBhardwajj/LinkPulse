## Description
<!-- What does this PR do and why? Link the related issue if applicable. -->

Closes #

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to change)
- [ ] Documentation update
- [ ] Refactor / cleanup (no behavior change)
- [ ] Performance improvement
- [ ] Security improvement

## Changes Made
<!-- List the key files and what changed in each. -->

- `internal/...`: 
- `docs/...`: 

## Testing
<!-- Describe how you tested the change. -->

- [ ] New unit tests added
- [ ] Existing tests updated
- [ ] Manual testing performed

```bash
# Commands used to test
go test ./...
go test -race ./...
go vet ./...
```

## Checklist
- [ ] Code follows Clean Architecture boundaries (handler → service → repository)
- [ ] No sensitive data (passwords, tokens, secrets) logged
- [ ] Error responses use RFC 7807 problem+json format
- [ ] New endpoints documented in `docs/swagger.json`
- [ ] `go mod tidy` run — `go.mod` and `go.sum` are up to date
- [ ] `gofmt -w .` applied — no formatting diffs
- [ ] `go vet ./...` passes with zero warnings
- [ ] `go test -race ./...` passes with zero failures
- [ ] `go build ./...` succeeds

## Screenshots / API Output (if applicable)
<!-- Paste relevant curl output or response bodies here. Redact credentials. -->
