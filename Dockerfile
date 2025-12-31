# Multi-stage Dockerfile for remarkable-sync
# Builds a minimal Alpine-based image with Tesseract OCR

# Build stage
FROM golang:1.25-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    tesseract-ocr-dev \
    leptonica-dev \
    pkgconfig

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary with CGO enabled
# pkg-config is available in the build image
RUN CGO_ENABLED=1 \
    CGO_CFLAGS="$(pkg-config --cflags tesseract lept)" \
    CGO_LDFLAGS="$(pkg-config --libs tesseract lept)" \
    go build -ldflags="-s -w" -o remarkable-sync ./cmd/remarkable-sync

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    leptonica \
    ca-certificates \
    tzdata

# Create non-root user
RUN adduser -D -u 1000 -h /home/remarkable remarkable

# Create directories with correct permissions
RUN mkdir -p /home/remarkable/.rmapi /output && \
    chown -R remarkable:remarkable /home/remarkable /output

# Copy binary from builder
COPY --from=builder /build/remarkable-sync /usr/local/bin/remarkable-sync

# Switch to non-root user
USER remarkable
WORKDIR /home/remarkable

# Set up volumes
VOLUME ["/home/remarkable/.rmapi", "/output"]

# Default command
ENTRYPOINT ["remarkable-sync"]
CMD ["--help"]

# Metadata labels
LABEL org.opencontainers.image.title="remarkable-sync"
LABEL org.opencontainers.image.description="Sync documents from reMarkable tablet with OCR text layer generation"
LABEL org.opencontainers.image.url="https://github.com/platinummonkey/remarkable-sync"
LABEL org.opencontainers.image.source="https://github.com/platinummonkey/remarkable-sync"
LABEL org.opencontainers.image.vendor="platinummonkey"
LABEL org.opencontainers.image.licenses="MIT"
