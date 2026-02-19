# API Reference: coin-futures-websocket

This document describes the HTTP and WebSocket API exposed by the coin-futures-websocket service.

---

## HTTP Endpoints

### Health Check

```
GET /health
```

Returns the current server status and number of active WebSocket connections.

**Response** `200 OK`:
```json
{"status": "ok", "connections": 42}
```

---

### Prometheus Metrics

```
GET /metrics
```

Returns Prometheus-formatted metrics. Standard Prometheus scrape endpoint.

**Metrics exposed**:

| Metric | Type | Description |
|--------|------|-------------|
| `centrifuge_connections_total` | Counter | Total connections by node |
| `centrifuge_connections_active` | Gauge | Currently active connections |
| `centrifuge_connections_failed_total` | Counter | Failed connections by node and reason |
| `centrifuge_subscriptions_total` | Counter | Total subscriptions by node and channel |
| `centrifuge_subscriptions_active` | Gauge | Currently active subscriptions |
| `centrifuge_messages_published_total` | Counter | Messages published by node and channel |

---

### WebSocket Connection

```
GET /connection  (WebSocket upgrade)
```

WebSocket endpoint using the [Centrifuge protocol](https://centrifugal.dev/docs/transports/websocket). Clients must use a Centrifuge SDK or implement the Centrifuge protocol directly.

---

## Authentication

Authentication uses JWT passed in the Centrifuge Connect command's `token` field.

**JWT requirements**:
- The `sub` (subject) claim must contain the Ajaib user ID (numeric string, e.g. `"130010505"`)
- The token is validated on every connection

**Fallback**: For backward compatibility, the JWT may also be passed in the `X-Socket-Authorization` HTTP header during the WebSocket upgrade handshake.

**Connection flow**:
1. Client sends Connect command with JWT token
2. Server parses `sub` claim to extract `ajaib_id`
3. Server resolves `cfx_user_id` via coin-cfx-adapter
4. Server fetches user's `quote_preference` via coin-setting
5. Connection is established with user metadata stored in session

---

## Channels

### Format

```
user:{ajaib_id}:{type}
```

| Part | Description |
|------|-------------|
| `ajaib_id` | Numeric Ajaib user ID (1–10 digits) |
| `type` | `margin` or `position` |

**Examples**:
```
user:130010505:margin
user:130010505:position
```

### Authorization

Users can only subscribe to their own channels. The `ajaib_id` in the channel name must match the `sub` claim from the connected JWT. Subscribing to another user's channel returns error `4001`.

---

## Message Payloads

Messages are delivered as Centrifuge publications. The `data` field of each publication contains a JSON object.

### Margin (`user:{ajaib_id}:margin`)

Published when a user's margin account state changes.

```json
{
  "timestamp": 1708300000000,
  "cfx_user_id": "cfx-user-abc",
  "asset": "USDT",
  "total_position_value": 10000.0,
  "margin_balance": 5000.0,
  "order_margin": 200.0,
  "effective_leverage": 2.0,
  "maintenance_margin": 100.0,
  "unrealized_pnl": -50.0,
  "available_margin": 4750.0,
  "wallet_balance": 5000.0,
  "margin_ratio": 0.02,
  "withdrawable_margin": 4500.0
}
```

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | int64 | Event time in milliseconds |
| `cfx_user_id` | string | CFX internal user identifier |
| `asset` | string | Quote asset (e.g. `USDT`, `IDR`) |
| `total_position_value` | float64 | Total value of open positions |
| `margin_balance` | float64 | Current margin balance |
| `order_margin` | float64 | Margin locked in open orders |
| `effective_leverage` | float64 | Current effective leverage |
| `maintenance_margin` | float64 | Minimum margin required |
| `unrealized_pnl` | float64 | Unrealized profit/loss |
| `available_margin` | float64 | Margin available for new orders |
| `wallet_balance` | float64 | Total wallet balance |
| `margin_ratio` | float64 | Margin ratio (maintenance / balance) |
| `withdrawable_margin` | float64 | Amount that can be withdrawn |

### Position (`user:{ajaib_id}:position`)

Published when a user's futures position changes.

```json
{
  "timestamp": 1708300000000,
  "cfx_user_id": "cfx-user-abc",
  "symbol": "BTCUSDT",
  "size": 0.5,
  "value": 25000.0,
  "leverage": 10,
  "entry_price": 48000.0,
  "mark_price": 50000.0,
  "liquidation_price": 43000.0,
  "maintenance_margin": 125.0,
  "realised_pnl": 100.0,
  "unrealised_pnl": 1000.0,
  "deleverage_percentile": 0.75,
  "risk_limit": 1000000,
  "open_order_buy_cost": 0.0,
  "open_order_sell_cost": 0.0,
  "initial_margin_requirement": 2500.0,
  "updated_time": 1708300000000,
  "open_order_buy_quantity": 0.0,
  "open_order_sell_quantity": 0.0,
  "order_margin": 0.0
}
```

| Field | Type | Description |
|-------|------|-------------|
| `timestamp` | int64 | Event time in milliseconds |
| `cfx_user_id` | string | CFX internal user identifier |
| `symbol` | string | Futures contract symbol (e.g. `BTCUSDT`) |
| `size` | float64 | Position size (negative = short) |
| `value` | float64 | Position notional value |
| `leverage` | int | Leverage multiplier |
| `entry_price` | float64 | Average entry price |
| `mark_price` | float64 | Current mark price |
| `liquidation_price` | float64 | Estimated liquidation price |
| `maintenance_margin` | float64 | Maintenance margin required |
| `realised_pnl` | float64 | Realized profit/loss |
| `unrealised_pnl` | float64 | Unrealized profit/loss |
| `deleverage_percentile` | float64 | ADL queue percentile (0–1) |
| `risk_limit` | int64 | Position risk limit |
| `open_order_buy_cost` | float64 | Cost of open buy orders |
| `open_order_sell_cost` | float64 | Cost of open sell orders |
| `initial_margin_requirement` | float64 | Initial margin for this position |
| `updated_time` | int64 | Last update time in milliseconds |
| `open_order_buy_quantity` | float64 | Open buy order quantity |
| `open_order_sell_quantity` | float64 | Open sell order quantity |
| `order_margin` | float64 | Margin locked in orders |

> **Note**: Numeric fields are denominated in the user's `quote_preference` asset (e.g. `USDT` or `IDR`). Currency conversion is applied server-side before publication.

---

## Error Codes

Errors are returned as Centrifuge error objects with a `code` and `message` field.

### Non-terminal Errors (4000–4499)

Client should reconnect after these errors.

| Code | Name | Description |
|------|------|-------------|
| 4000 | Bad Request | Invalid request format |
| 4001 | Channel Not Found | Channel format invalid or user not authorized for this channel |
| 4002 | Already Subscribed | Client is already subscribed to this channel |
| 4003 | Not Subscribed | Client is not subscribed to this channel |
| 4004 | Subscription Limit | Subscription limit exceeded |
| 4100 | Unauthorized | JWT is missing, invalid, or expired |
| 4200 | Connection Limit | Too many concurrent connections for this user |

### Terminal Errors (4500–4999)

Client should **not** auto-reconnect after these errors. Operator investigation required.

| Code | Name | Description |
|------|------|-------------|
| 4500 | Internal Error | Unexpected server error |
| 4501 | CFX User Resolution Error | Failed to resolve Ajaib ID to CFX user ID via coin-cfx-adapter |
| 4502 | User Preference Error | Failed to fetch quote preference via coin-setting |
| 4503 | Service Unavailable | Downstream service unavailable |

---

## Client Examples

### Go (centrifuge-go)

See [`examples/client/main.go`](../examples/client/main.go) for a complete example.

### JavaScript (centrifuge-js)

```javascript
import { Centrifuge } from 'centrifuge';

const client = new Centrifuge('ws://localhost:8009/connection', {
  token: 'your_jwt_token'
});

const sub = client.newSubscription('user:130010505:margin');

sub.on('publication', (ctx) => {
  console.log('Margin update:', ctx.data);
});

sub.subscribe();
client.connect();
```
