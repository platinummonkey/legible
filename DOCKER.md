# Docker Guide for remarkable-sync

This guide explains how to use remarkable-sync with Docker and Ollama for OCR processing.

## System Requirements

Before running remarkable-sync with Docker, ensure your system meets these requirements:

- **Disk Space**: 8GB+ available (Ollama model is ~4-7GB)
- **RAM**: 4GB+ recommended for OCR processing
- **CPU**: Modern CPU with decent single-thread performance
- **Docker**: Docker Engine 20.10+ or Docker Desktop

## Quick Start

### 1. Using Docker Compose (Recommended)

The easiest way to run remarkable-sync is with Docker Compose:

```bash
# Download the example docker-compose.yml
curl -O https://raw.githubusercontent.com/platinummonkey/remarkable-sync/main/examples/docker-compose.yml

# Authenticate with reMarkable cloud (one-time)
docker-compose run --rm remarkable-sync auth

# Start the daemon
docker-compose up -d

# View logs
docker-compose logs -f remarkable-sync

# Stop the daemon
docker-compose down
```

### 2. Using Docker Run

For more control, you can use `docker run` directly:

```bash
# Create volumes
docker volume create remarkable-credentials
docker volume create ollama-models

# Authenticate (one-time)
docker run --rm -it \
  -v remarkable-credentials:/home/remarkable/.rmapi \
  ghcr.io/platinummonkey/remarkable-sync:latest auth

# Run daemon
docker run -d \
  --name remarkable-sync \
  -v remarkable-credentials:/home/remarkable/.rmapi \
  -v ./output:/output \
  -v ollama-models:/home/remarkable/.ollama/models \
  -e OCR_MODEL=llava \
  ghcr.io/platinummonkey/remarkable-sync:latest \
  daemon --interval 1h --output /output
```

## Configuration

### Environment Variables

Configure Ollama and OCR behavior with these environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama API endpoint |
| `OCR_MODEL` | `llava` | Vision model for OCR (llava, mistral, etc.) |
| `OLLAMA_MODELS` | `/home/remarkable/.ollama/models` | Model storage directory |

### Volumes

Mount these volumes for persistent data:

| Volume | Purpose | Required |
|--------|---------|----------|
| `/home/remarkable/.rmapi` | reMarkable credentials | Yes |
| `/output` | Synced PDFs output | Yes |
| `/home/remarkable/.ollama/models` | Ollama models cache | No (but recommended) |

## OCR Models

### Available Models

The container comes with **llava** pre-downloaded, but you can use other models:

| Model | Size | Accuracy | Speed | Recommended For |
|-------|------|----------|-------|-----------------|
| `llava` | ~4GB | Good | Fast | General handwriting (default) |
| `llava:13b` | ~7GB | Excellent | Slower | Best accuracy |
| `mistral` | ~4GB | Good | Fast | Alternative to llava |

### Changing Models

To use a different model:

```yaml
# docker-compose.yml
services:
  remarkable-sync:
    environment:
      - OCR_MODEL=llava:13b  # Use larger, more accurate model
```

Or with `docker run`:

```bash
docker run -d \
  -e OCR_MODEL=llava:13b \
  # ... other options
```

The model will be automatically downloaded on first run.

## Advanced Configuration

### Using External Ollama Service

If you want to run Ollama separately (e.g., shared across multiple containers):

```yaml
# docker-compose.yml
version: '3.8'

services:
  ollama:
    image: ollama/ollama:latest
    container_name: ollama
    restart: unless-stopped
    volumes:
      - ollama-models:/root/.ollama
    ports:
      - "11434:11434"

  remarkable-sync:
    image: ghcr.io/platinummonkey/remarkable-sync:latest
    depends_on:
      - ollama
    environment:
      - OLLAMA_HOST=http://ollama:11434
      - OCR_MODEL=llava
    volumes:
      - ./credentials:/home/remarkable/.rmapi
      - ./output:/output
    command: daemon --interval 1h --output /output

volumes:
  ollama-models:
```

### Resource Limits

Adjust resource limits based on your model and usage:

```yaml
services:
  remarkable-sync:
    deploy:
      resources:
        limits:
          cpus: '2.0'      # Max CPU cores
          memory: 4G       # Max RAM
        reservations:
          cpus: '0.5'      # Min CPU cores
          memory: 1G       # Min RAM
```

### Custom Sync Schedule

Control how often documents are synced:

```bash
# Sync every 30 minutes
command: daemon --interval 30m --output /output

# Sync every 6 hours
command: daemon --interval 6h --output /output

# One-time sync (not daemon mode)
command: sync --output /output
```

### Label Filtering

Sync only documents with specific labels:

```bash
command: daemon --interval 1h --output /output --labels "work,personal"
```

## Building from Source

To build the Docker image locally:

```bash
# Clone the repository
git clone https://github.com/platinummonkey/remarkable-sync.git
cd remarkable-sync

# Build with GoReleaser for multi-platform support
goreleaser build --snapshot --clean

# Build Docker image
docker build -t remarkable-sync:local .

# Run your local build
docker run -d \
  -v ./credentials:/home/remarkable/.rmapi \
  -v ./output:/output \
  -v ollama-models:/home/remarkable/.ollama/models \
  remarkable-sync:local \
  daemon --interval 1h --output /output
```

## Troubleshooting

### Ollama Not Starting

**Symptoms**: Container logs show "Ollama failed to start"

**Solutions**:
1. Check available disk space (need 8GB+)
2. Check available RAM (need 4GB+)
3. Try increasing start-period in health check
4. Check Docker logs: `docker logs remarkable-sync`

### Model Download Fails

**Symptoms**: "Failed to download model" or "Model not found"

**Solutions**:
1. Check internet connection
2. Verify model name is correct (case-sensitive)
3. Try downloading manually:
   ```bash
   docker exec -it remarkable-sync ollama pull llava
   ```
4. Check Ollama service status:
   ```bash
   docker exec -it remarkable-sync curl http://localhost:11434/api/tags
   ```

### OCR Not Working

**Symptoms**: PDFs generated but not searchable

**Solutions**:
1. Verify Ollama is running:
   ```bash
   docker exec -it remarkable-sync curl http://localhost:11434/api/tags
   ```
2. Check model is downloaded:
   ```bash
   docker exec -it remarkable-sync ollama list
   ```
3. View logs for OCR errors:
   ```bash
   docker logs remarkable-sync | grep -i ocr
   ```
4. Try a different model:
   ```bash
   docker-compose down
   # Edit docker-compose.yml to change OCR_MODEL
   docker-compose up -d
   ```

### Out of Memory

**Symptoms**: Container crashes or gets killed

**Solutions**:
1. Increase Docker memory limit
2. Use a smaller model (llava instead of llava:13b)
3. Adjust resource limits in docker-compose.yml
4. Close other applications to free RAM

### Slow Performance

**Symptoms**: OCR takes a long time per page

**Solutions**:
1. This is normal (2-5s per page expected)
2. Use a faster model (llava instead of llava:13b)
3. Increase CPU allocation
4. Consider using external Ollama on a more powerful machine
5. Process pages in parallel (if supported by your workflow)

## Security Considerations

### Non-Root User

The container runs as a non-root user (`remarkable`) for security:

```bash
# Verify user
docker exec remarkable-sync whoami
# Output: remarkable
```

### Credential Storage

reMarkable API credentials are stored in `/home/remarkable/.rmapi`:

- **Mount as volume**: Credentials persist across container restarts
- **Keep private**: Don't commit credentials to version control
- **Backup**: Save credentials volume before major changes

### Network Security

The container doesn't expose any ports by default:

- Ollama runs on localhost only (not accessible externally)
- No incoming network connections required
- Only outbound connections: reMarkable API and Ollama model downloads

## Performance Tuning

### Model Selection

Choose the right model for your needs:

```yaml
# Fast but less accurate
environment:
  - OCR_MODEL=llava

# Slow but more accurate
environment:
  - OCR_MODEL=llava:13b

# Alternative vision model
environment:
  - OCR_MODEL=mistral
```

### Parallel Processing

Ollama can process multiple requests concurrently:

```yaml
environment:
  - OLLAMA_NUM_PARALLEL=2
  - OLLAMA_MAX_LOADED_MODELS=1
```

### Disk Usage

Monitor disk usage for Ollama models:

```bash
# Check model sizes
docker exec remarkable-sync du -sh /home/remarkable/.ollama/models/*

# Clean up unused models
docker exec remarkable-sync ollama rm old-model-name
```

## Monitoring

### Health Checks

The container includes health checks for both Ollama and remarkable-sync:

```bash
# Check health status
docker inspect --format='{{.State.Health.Status}}' remarkable-sync

# View health check logs
docker inspect --format='{{range .State.Health.Log}}{{.Output}}{{end}}' remarkable-sync
```

### Logs

View comprehensive logs:

```bash
# All logs
docker logs remarkable-sync

# Follow logs in real-time
docker logs -f remarkable-sync

# Last 100 lines
docker logs --tail 100 remarkable-sync

# Logs since 1 hour ago
docker logs --since 1h remarkable-sync
```

### Metrics

Monitor resource usage:

```bash
# Real-time stats
docker stats remarkable-sync

# Current resource usage
docker exec remarkable-sync ps aux
docker exec remarkable-sync df -h
```

## Migration from Tesseract

If you were using an older version with Tesseract:

### Key Changes

| Aspect | Old (Tesseract) | New (Ollama) |
|--------|----------------|--------------|
| **OCR Engine** | Tesseract + Leptonica | Ollama vision models |
| **Dependencies** | System packages | HTTP API |
| **Accuracy** | 40-60% (handwriting) | 85-95% (handwriting) |
| **Speed** | ~0.5-1s/page | ~2-5s/page |
| **Image Size** | ~200MB | ~4-7GB (with model) |

### Migration Steps

1. **Pull new image**: `docker pull ghcr.io/platinummonkey/remarkable-sync:latest`
2. **Update docker-compose.yml**: Add Ollama volumes and environment variables
3. **First run**: Model will download automatically (may take several minutes)
4. **Test**: Verify OCR quality on test documents
5. **Clean up**: Remove old Tesseract-based images

## Support

- **Issues**: https://github.com/platinummonkey/remarkable-sync/issues
- **Discussions**: https://github.com/platinummonkey/remarkable-sync/discussions
- **Documentation**: https://github.com/platinummonkey/remarkable-sync/blob/main/README.md

## License

Part of remarkable-sync project. See project LICENSE for details.
