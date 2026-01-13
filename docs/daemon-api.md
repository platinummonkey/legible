# Daemon Status API

The Legible daemon exposes an HTTP API for monitoring and control when running with the `--health-addr` flag.

## Overview

The API provides endpoints for:
- Health checking (for monitoring systems)
- Status monitoring (for UI applications like the menu bar app)
- Sync control (trigger/cancel operations)

## Configuration

Enable the API by starting the daemon with the `--health-addr` flag:

```bash
legible daemon --health-addr localhost:8080
```

**Security Note**: The API currently has no authentication. Bind to `localhost` only for local access, or use `127.0.0.1:8080` to restrict to local connections only.

## Endpoints

### Health & Readiness

#### `GET /health`

Basic health check endpoint.

**Response**: `200 OK`
```
OK
```

**Use case**: Monitoring systems (e.g., Kubernetes liveness probes)

---

#### `GET /ready`

Readiness check endpoint.

**Response**: `200 OK`
```
OK
```

**Use case**: Monitoring systems (e.g., Kubernetes readiness probes)

---

### Status Monitoring

#### `GET /status`

Returns the current daemon status including sync state, progress, and history.

**Response**: `200 OK`

```json
{
  "state": "idle",
  "last_sync_time": "2026-01-13T18:30:00Z",
  "next_sync_time": "2026-01-13T18:35:00Z",
  "sync_duration": 45000000000,
  "error_message": "",
  "current_sync": null,
  "last_sync_result": {
    "start_time": "2026-01-13T18:29:15Z",
    "end_time": "2026-01-13T18:30:00Z",
    "duration": 45000000000,
    "total_documents": 150,
    "processed_documents": 5,
    "success_count": 5,
    "failure_count": 0,
    "skipped_count": 145
  },
  "uptime_seconds": 3600
}
```

**Fields**:

- `state` (string): Current sync state
  - `"idle"` - Daemon is running, waiting for next sync
  - `"syncing"` - Sync operation in progress
  - `"error"` - Last sync failed

- `last_sync_time` (string, nullable): ISO 8601 timestamp of last sync attempt

- `next_sync_time` (string, nullable): ISO 8601 timestamp of next scheduled sync

- `sync_duration` (number, nullable): Duration of last sync in nanoseconds

- `error_message` (string): Error message if state is "error"

- `current_sync` (object, nullable): Information about in-progress sync
  - `start_time` (string): When the current sync started
  - `documents_total` (number): Total documents to process
  - `documents_processed` (number): Documents processed so far
  - `current_document` (string): Document currently being processed
  - `stage` (string): Current processing stage (downloading, converting, ocr, enhancing)

- `last_sync_result` (object, nullable): Summary of last completed sync
  - `start_time` (string): When the sync started
  - `end_time` (string): When the sync completed
  - `duration` (number): Duration in nanoseconds
  - `total_documents` (number): Total documents checked
  - `processed_documents` (number): Documents that were processed
  - `success_count` (number): Successfully processed documents
  - `failure_count` (number): Failed documents
  - `skipped_count` (number): Documents skipped (no changes)

- `uptime_seconds` (number): How long the daemon has been running

**Example - Idle State**:
```json
{
  "state": "idle",
  "last_sync_time": "2026-01-13T18:30:00Z",
  "next_sync_time": "2026-01-13T18:35:00Z",
  "sync_duration": 45000000000,
  "last_sync_result": {
    "start_time": "2026-01-13T18:29:15Z",
    "end_time": "2026-01-13T18:30:00Z",
    "duration": 45000000000,
    "total_documents": 150,
    "processed_documents": 5,
    "success_count": 5,
    "failure_count": 0,
    "skipped_count": 145
  },
  "uptime_seconds": 3600
}
```

**Example - Syncing State**:
```json
{
  "state": "syncing",
  "last_sync_time": "2026-01-13T18:35:00Z",
  "next_sync_time": "2026-01-13T18:40:00Z",
  "current_sync": {
    "start_time": "2026-01-13T18:35:00Z",
    "documents_total": 150,
    "documents_processed": 42,
    "current_document": "My Notes.rmdoc",
    "stage": "ocr"
  },
  "uptime_seconds": 3900
}
```

**Example - Error State**:
```json
{
  "state": "error",
  "last_sync_time": "2026-01-13T18:35:00Z",
  "next_sync_time": "2026-01-13T18:40:00Z",
  "sync_duration": 1200000000,
  "error_message": "authentication failed: invalid token",
  "uptime_seconds": 3900
}
```

---

### Control Endpoints

#### `POST /api/sync/trigger`

Triggers an immediate sync operation (bypasses the scheduled interval).

**Request**: `POST` with empty body

**Response**: `202 Accepted` (if accepted) or `409 Conflict` (if sync already running)

```json
{
  "success": true,
  "message": "Sync triggered successfully"
}
```

**Error Response** (sync already in progress):
```json
{
  "success": false,
  "message": "Sync already in progress"
}
```

**Current Status**: ⚠️ Not yet fully implemented - endpoint exists but manual triggering mechanism is incomplete.

---

#### `POST /api/sync/cancel`

Attempts to cancel an in-progress sync operation.

**Request**: `POST` with empty body

**Response**: `200 OK` (if canceled) or `409 Conflict` (if no sync running)

```json
{
  "success": true,
  "message": "Sync canceled"
}
```

**Error Response** (no sync in progress):
```json
{
  "success": false,
  "message": "No sync in progress to cancel"
}
```

**Current Status**: ⚠️ Not yet implemented - endpoint exists but cancellation mechanism is incomplete.

---

## Usage Examples

### Check Daemon Status

```bash
curl http://localhost:8080/status | jq
```

### Monitor Sync Progress

```bash
# Poll every 2 seconds
watch -n 2 'curl -s http://localhost:8080/status | jq ".state, .current_sync"'
```

### Trigger Manual Sync

```bash
curl -X POST http://localhost:8080/api/sync/trigger
```

### Cancel Running Sync

```bash
curl -X POST http://localhost:8080/api/sync/cancel
```

---

## Integration with Menu Bar App

The menu bar application uses this API to:

1. **Poll for status** every 2-5 seconds via `GET /status`
2. **Update icon color** based on `state` field:
   - Green: `"idle"`
   - Yellow: `"syncing"`
   - Red: `"error"`
3. **Display status info** in menu (last sync time, document counts, errors)
4. **Trigger sync** via `POST /api/sync/trigger` (when implemented)
5. **Cancel sync** via `POST /api/sync/cancel` (when implemented)

Example polling code:

```go
ticker := time.NewTicker(3 * time.Second)
for range ticker.C {
    resp, err := http.Get("http://localhost:8080/status")
    if err != nil {
        // Show red icon, daemon not running
        continue
    }
    defer resp.Body.Close()

    var status Status
    if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
        // Handle error
        continue
    }

    // Update UI based on status.State
    switch status.State {
    case "idle":
        app.SetStatusIdle()
    case "syncing":
        app.SetStatusSyncing()
    case "error":
        app.SetStatusError(status.ErrorMessage)
    }
}
```

---

## Future Enhancements

### Planned Features

1. **Authentication**: Add API key or token-based auth for control endpoints
2. **WebSocket support**: Real-time status updates instead of polling
3. **Manual sync trigger**: Implement actual trigger mechanism
4. **Sync cancellation**: Implement context-based cancellation
5. **Detailed progress**: Per-document progress updates
6. **Configuration endpoint**: View/update daemon configuration via API
7. **Log streaming**: Stream daemon logs via `/api/logs` endpoint

### Security Considerations

For production use:

1. **Bind to localhost only**: `--health-addr localhost:8080` or `127.0.0.1:8080`
2. **Add authentication**: Implement API key auth for control endpoints
3. **Use Unix socket**: Alternative to TCP for local-only access (future)
4. **Rate limiting**: Prevent abuse of trigger endpoint

---

## Architecture Decision: HTTP vs Unix Socket

We chose HTTP over Unix sockets for the following reasons:

**Advantages of HTTP**:
- ✅ Cross-platform (Windows, macOS, Linux)
- ✅ Easy to test with curl/browser
- ✅ Language-agnostic clients
- ✅ Can be exposed remotely if needed (future)
- ✅ Standard health check format for monitoring systems
- ✅ Familiar REST semantics

**Advantages of Unix Socket** (not chosen):
- ✅ Slightly lower latency
- ✅ File permission-based security
- ✅ No port conflicts
- ❌ macOS/Linux only
- ❌ Harder to test manually
- ❌ Requires special client code

**Conclusion**: HTTP provides better compatibility and ease of use. For local-only access, binding to `localhost` provides adequate security.

---

## Troubleshooting

### "Connection refused" when accessing API

**Problem**: `curl: (7) Failed to connect to localhost port 8080: Connection refused`

**Solutions**:
1. Check daemon is running with `--health-addr` flag
2. Verify the port matches: `legible daemon --health-addr :8080`
3. Check if another process is using the port: `lsof -i :8080`

### Status shows "error" state

**Problem**: Status endpoint returns `"state": "error"`

**Solutions**:
1. Check `error_message` field for details
2. Check daemon logs: `journalctl -u legible -f` (systemd) or app logs
3. Verify authentication: `legible auth` to re-authenticate
4. Check network connectivity to reMarkable API

### Status API returns 404

**Problem**: `/status` returns 404 Not Found

**Solutions**:
1. Verify daemon version supports status API (v1.3.0+)
2. Confirm `--health-addr` flag is set
3. Try `/health` endpoint to verify server is running

---

## Version History

- **v1.3.0** (2026-01-13): Initial status API implementation
  - Added `/status` endpoint
  - Added `/api/sync/trigger` stub
  - Added `/api/sync/cancel` stub
  - Thread-safe status tracking
