# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git and certificates
RUN apk add --no-cache git ca-certificates

# Copy go.mod and go.sum first for dependency caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build compile binary (statically linked, size optimized)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o linkpulse cmd/api/main.go

# Stage 2: Minimal runtime image
FROM alpine:3.19

# Create a non-privileged user and group for container runtime safety
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Install ca-certificates and curl for health checks
RUN apk add --no-cache ca-certificates curl

# Copy the built binary from builder stage
COPY --from=builder /app/linkpulse .

# Copy environment template
COPY .env.example .env

# Adjust folder ownership to the non-privileged user
RUN chown -R appuser:appgroup /app

# Switch to non-root execution context
USER appuser

# Expose server port
EXPOSE 8080

# Execute server binary
ENTRYPOINT ["./linkpulse"]
