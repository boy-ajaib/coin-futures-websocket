# Migration Guide: Gorilla WebSocket → Centrifuge

This document describes the migration from `gorilla/websocket` to `centrifuge-go` completed in early 2026.

## Overview

The migration replaced a custom WebSocket implementation using `gorilla/websocket` with the production-ready `centrifuge-go` library. This change standardizes WebSocket implementation across projects and provides built-in features like automatic reconnection, presence, and history.

## Breaking Changes

### Protocol Change

**Before (Custom JSON Protocol)**:
```json
{"type":"subscribe","channel":"user:130010505:margin","id":"sub-1"}
```

**After (Centrifuge Protocol)**:
```json
{"id":1,"subscribe":{"user:130010505:margin":{}}}
```

Clients must use the Centrifuge protocol or the Centrifuge client SDKs.

### Connection Endpoint

- **Before**: `ws://localhost:8009/ws`
- **After**: `ws://localhost:8009/connection`

### Authentication

**Before**: JWT only in `X-Socket-Authorization` header

**After**: JWT in either:
- Centrifuge token field in connect command
- `X-Socket-Authorization` header (for backward compatibility)

## Architecture Changes

### Server Implementation

| Aspect | Before (Gorilla) | After (Centrifuge) |
|--------|------------------|-------------------|
| Server | Custom `http.Server` with upgrade | `centrifuge.Node` |
| Client Management | Custom `Hub` and `Client` structs | Built-in connection handling |
| Protocol | Custom JSON messages | Centrifuge binary protocol |
| Channels | Custom map | Built-in channel engine |
| Handlers | Manual read/write pumps | Event handlers (OnConnect, OnSubscribe, etc.) |

### Code Changes

**Server Creation**:
```go
// Before
hub := server.NewHub()
wsServer := server.NewServer(hub, config)

// After
centrifugeServer := server.NewCentrifugeServer(&cfg.Centrifuge, logger)
```

**Broadcasting**:
```go
// Before
hub.Broadcast(channel, data)

// After
node.Publish(channel, data)
```

## Migration Timeline

| Milestone | Description | Status |
|-----------|-------------|--------|
| M1 | Project Setup & Dependencies | ✅ Complete |
| M2 | Centrifuge Server Foundation | ✅ Complete |
| M3 | Authentication & User Resolution | ✅ Complete |
| M4 | Channel Management & Validation | ✅ Complete |
| M5 | Kafka Integration & Broadcasting | ✅ Complete |
| M6 | Protocol Migration | ✅ Complete |
| M7 | Configuration & Logging | ✅ Complete |
| M8 | Testing & Validation | ✅ Complete |
| M9 | Code Cleanup & Documentation | ✅ Complete |
| M10 | Merge & Deployment | Pending |

## Files Removed

The following files were removed during migration:

- `internal/websocket/server/hub.go` - Custom hub implementation
- `internal/websocket/server/client.go` - Custom client implementation
- `internal/websocket/server/server.go` - Custom server implementation
- `internal/websocket/handler/` - Old handler directory
- `internal/websocket/protocol/` - Custom protocol package

## Files Added

The following files were added during migration:

- `internal/websocket/server/centrifuge_server.go` - Centrifuge wrapper
- `internal/websocket/server/handlers.go` - Centrifuge event handlers
- `internal/websocket/server/errors.go` - Centrifuge-compatible error codes
- `internal/websocket/server/logger.go` - Structured logging adapter
- `internal/websocket/server/metrics.go` - Prometheus metrics
- `internal/auth/jwt.go` - JWT parsing
- `internal/auth/middleware.go` - HTTP middleware
- `examples/client/main.go` - Example Centrifuge Go client
- `CLAUDE.md` - Project documentation for AI

## Error Code Mapping

| Old Code | New Code | Description |
|----------|----------|-------------|
| Custom | 4000 | Bad Request |
| Custom | 4001 | Channel Not Found |
| Custom | 4100 | Unauthorized |
| Custom | 4500 | Internal Error |
| Custom | 4501 | CFX User Resolution Error |
| Custom | 4502 | User Preference Error |

See `internal/websocket/server/errors.go` for complete mapping.

## Client Migration

### JavaScript Clients

**Before** (using native WebSocket):
```javascript
const ws = new WebSocket('ws://localhost:8009/ws', ['protocol']);
ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'subscribe',
    channel: 'user:130010505:margin',
    id: 'sub-1'
  }));
};
```

**After** (using Centrifuge JS SDK):
```javascript
import Centrifuge from 'centrifuge';

const client = new Centrifuge('ws://localhost:8009/connection', {
  token: 'your_jwt_token'
});

client.on('connected', () => {
  const sub = client.subscribe('user:130010505:margin');
  sub.on('publication', (ctx) => {
    console.log('Received:', ctx.data);
  });
});

client.connect();
```

### Go Clients

See `examples/client/main.go` for a complete example using `centrifuge-go`.

## Testing

Unit tests were added for:
- JWT parsing (`internal/auth/jwt_test.go`)
- Kafka broadcasting (`internal/kafka/broadcaster_test.go`)
- Channel parsing (`internal/websocket/channel/types_test.go`)
- Server handlers (`internal/websocket/server/handlers_test.go`)

Run tests:
```bash
go test ./... -v -cover
```

## Configuration Changes

Added `centrifuge` section to config:

```yaml
centrifuge:
    node_name: coin-futures-ws
    namespace: coin-futures
    log_level: info
    client_insecure: false
    client_announce: true
    presence: false
    join_leave: false
    history_size: 0
    history_ttl_seconds: 0
    force_recovery: false
```

Removed obsolete fields from `websocket_server`:
- `no_command_timeout_ms` (handled by Centrifuge internally)

## Metrics

New Prometheus metrics endpoint at `/metrics`:

- `centrifuge_connections_total` - Total connections
- `centrifuge_connections_active` - Active connections
- `centrifuge_subscriptions_total` - Total subscriptions
- `centrifuge_messages_published_total` - Messages published

## Rollback Plan

If issues arise post-deployment:

1. Revert to feature branch before migration
2. Deploy previous version using Gorilla WebSocket
3. Investigate issues in development environment
4. Client applications can support both protocols during transition period if needed

## Future Enhancements

With Centrifuge, the following features are now possible with minimal changes:

1. **Presence**: Show which users are online
2. **History**: Allow clients to recover missed messages
3. **RPC**: Add bidirectional RPC calls
4. **Redis Scaling**: Add Redis for horizontal scaling

These features were disabled in config (`presence: false`, `history_size: 0`) but can be enabled when needed.

## References

- [Centrifuge Documentation](https://centrifugal.dev/)
- [Centrifuge Go SDK](https://github.com/centrifugal/centrifuge-go)
- [Centrifuge JavaScript SDK](https://github.com/centrifugal/centrifuge-js)
