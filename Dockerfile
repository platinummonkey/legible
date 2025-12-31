# Runtime Dockerfile for remarkable-sync with Ollama
# Uses pre-built binaries from GoReleaser for multi-platform support
# GoReleaser provides binaries organized by $TARGETPLATFORM in build context

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    bash

# Install Ollama
RUN curl -fsSL https://ollama.ai/install.sh | sh

# Create non-root user
RUN adduser -D -u 1000 -h /home/remarkable remarkable

# Create directories with correct permissions
RUN mkdir -p /home/remarkable/.rmapi /home/remarkable/.ollama/models /output && \
    chown -R remarkable:remarkable /home/remarkable /output

# Copy pre-built binary from GoReleaser build context
# $TARGETPLATFORM is provided by Docker buildx (e.g., linux/amd64, linux/arm64)
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM}/remarkable-sync /usr/local/bin/remarkable-sync

# Build argument to select which OCR model to pre-download
# Options: mistral-small3.1, llava, llava:13b, none
# Set to "none" to skip pre-downloading (models can be mounted from host)
ARG OCR_MODEL=mistral-small3.1

# Copy entrypoint script
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# Switch to non-root user
USER remarkable
WORKDIR /home/remarkable

# Environment variables for Ollama configuration
ENV OLLAMA_HOST=http://localhost:11434
ENV OLLAMA_MODELS=/home/remarkable/.ollama/models

# Pre-download OCR model (controlled by OCR_MODEL build arg)
# This increases image size but avoids download on first run
# To skip pre-downloading, build with: --build-arg OCR_MODEL=none
# To use mistral-small3.1: --build-arg OCR_MODEL=mistral-small3.1
USER root
RUN if [ "$OCR_MODEL" != "none" ]; then \
        ollama serve & \
        OLLAMA_PID=$! && \
        sleep 5 && \
        ollama pull $OCR_MODEL && \
        kill $OLLAMA_PID && \
        wait $OLLAMA_PID 2>/dev/null || true; \
    fi
USER remarkable

# Set the OCR_MODEL environment variable to the build arg value
ENV OCR_MODEL=$OCR_MODEL

# Set up volumes
VOLUME ["/home/remarkable/.rmapi", "/home/remarkable/.ollama/models", "/output"]

# Health check for Ollama service
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:11434/api/tags || exit 1

# Use entrypoint script to start Ollama and remarkable-sync
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["remarkable-sync", "--help"]

# Metadata labels
LABEL org.opencontainers.image.title="remarkable-sync"
LABEL org.opencontainers.image.description="Sync documents from reMarkable tablet with Ollama-powered OCR"
LABEL org.opencontainers.image.url="https://github.com/platinummonkey/remarkable-sync"
LABEL org.opencontainers.image.source="https://github.com/platinummonkey/remarkable-sync"
LABEL org.opencontainers.image.vendor="platinummonkey"
LABEL org.opencontainers.image.licenses="MIT"
