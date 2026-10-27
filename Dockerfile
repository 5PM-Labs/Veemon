# syntax=docker/dockerfile:1

# Build stage — builds from the repo root because go.mod lives there.
# Build context must be the repo root (docker-compose.yml `context: .`).
FROM golang:1.26.9-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Cache downloaded dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build all three binaries.
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY migrations/ ./migrations/
COPY .golangci.yml ./

WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/migrate ./cmd/migrate \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker ./cmd/worker

# Final stage
FROM alpine:3.23

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata \
 && addgroup -S app && adduser -S -G app app

# Binaries and SQL migrations.
COPY --from=builder /out/server /out/migrate /out/worker /app/
COPY migrations/ ./migrations/

EXPOSE 3000 50051

USER app

# Liveness against the HTTP health endpoint (wget ships with alpine).
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:3000/health >/dev/null 2>&1 || exit 1

# Default to the API server; override the command to run ./migrate or ./worker.
CMD ["/app/server"]
