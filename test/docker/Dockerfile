# LLM Gateway Test Environment Dockerfile
# Build stage - use Aliyun mirror for faster builds in China
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Set Go proxy for China
ENV GOPROXY=https://goproxy.cn,direct
ENV CGO_ENABLED=0

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server binary
RUN GOOS=linux go build -ldflags="-w -s" -o /llmgw ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Copy binary from builder
COPY --from=builder /llmgw /app/llmgw

# Copy config file (will be overridden by volume mount in compose)
COPY test/docker/config.test.yaml /app/config.yaml

# Copy database migrations
COPY db/migrations /app/db/migrations

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=10s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/auth/login || exit 1

# Run the server
CMD ["/app/llmgw"]