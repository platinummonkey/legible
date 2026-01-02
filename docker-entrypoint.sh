#!/bin/bash
set -e

# Docker entrypoint script for legible
# This script starts Ollama service in the background and then runs legible

echo "Starting Ollama service..."

# Start Ollama in background
ollama serve &
OLLAMA_PID=$!

# Wait for Ollama to be ready
echo "Waiting for Ollama to start..."
MAX_ATTEMPTS=30
ATTEMPT=0

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        echo "Ollama is ready"
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo "ERROR: Ollama failed to start after $MAX_ATTEMPTS attempts"
        exit 1
    fi
    echo "Waiting for Ollama... (attempt $ATTEMPT/$MAX_ATTEMPTS)"
    sleep 1
done

# Check if OCR model is available
echo "Checking for OCR model: ${OCR_MODEL:-llava}"
if ! ollama list | grep -q "${OCR_MODEL:-llava}"; then
    echo "Model ${OCR_MODEL:-llava} not found. Downloading..."
    if ! ollama pull "${OCR_MODEL:-llava}"; then
        echo "WARNING: Failed to download model ${OCR_MODEL:-llava}"
        echo "OCR functionality may not work correctly"
    else
        echo "Model ${OCR_MODEL:-llava} downloaded successfully"
    fi
else
    echo "Model ${OCR_MODEL:-llava} is available"
fi

# Set up signal handling to gracefully shut down Ollama
trap 'echo "Shutting down Ollama..."; kill $OLLAMA_PID; wait $OLLAMA_PID 2>/dev/null || true; exit 0' SIGTERM SIGINT

# Execute the main command
echo "Starting legible..."
exec "$@"
