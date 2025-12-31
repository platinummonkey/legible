# Runtime Dockerfile for remarkable-sync
# Uses pre-built binaries from GoReleaser for multi-platform support
# GoReleaser provides binaries organized by $TARGETPLATFORM in build context

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

# Copy pre-built binary from GoReleaser build context
# $TARGETPLATFORM is provided by Docker buildx (e.g., linux/amd64, linux/arm64)
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/remarkable-sync /usr/local/bin/remarkable-sync

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
