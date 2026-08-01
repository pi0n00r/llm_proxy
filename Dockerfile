# Build stage
FROM golang:1.25-trixie AS builder

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o llm_proxy .

# Runtime stage
FROM debian:trixie-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Create the unprivileged runtime account and writable data directory.
WORKDIR /app

RUN groupadd --gid 10001 llm-proxy \
    && useradd --uid 10001 --gid 10001 --home-dir /nonexistent --no-create-home --shell /usr/sbin/nologin llm-proxy \
    && mkdir -p /app/data /app/config \
    && chown -R 10001:10001 /app

# Copy binary from builder
COPY --from=builder --chown=10001:10001 /build/llm_proxy .

USER 10001:10001

# Expose the compatibility-fork example port
EXPOSE 6666

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:6666/health || exit 1

# Run the application
ENTRYPOINT ["/app/llm_proxy"]
CMD ["-config", "/app/config/config.toml"]
