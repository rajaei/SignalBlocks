# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add git ca-certificates tzdata

# Copy Go module files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -a \
    -o /app/bin/signalblocks ./cmd/signalblocks

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add ca-certificates tzdata curl wget

# Copy the binary from builder
COPY --from=builder /app/bin/signalblocks /app/signalblocks

# Create non-root user
RUN addgroup -g 1000 signalblocks && \
    adduser -D -u 1000 -G signalblocks signalblocks

# Change ownership
RUN chown -R signalblocks:signalblocks /app

USER signalblocks

# Health check
HEALTHCHECK --interval=10s --timeout=5s --retries=3 --start-period=10s \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["/app/signalblocks"]
