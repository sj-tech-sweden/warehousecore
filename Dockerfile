# Multi-stage build for StorageCore
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o storagecore ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/storagecore .

# Copy migrations
COPY --from=builder /app/migrations ./migrations

# Copy frontend dist (if exists)
COPY --from=builder /app/web/dist ./web/dist 2>/dev/null || mkdir -p ./web/dist

# Create .env placeholder
RUN touch .env

EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/api/v1/health || exit 1

CMD ["./storagecore"]
