# Centrifuge Go Client Example

This example demonstrates how to connect to the Coin Futures WebSocket service using the [Centrifuge Go SDK](https://github.com/centrifugal/centrifuge-go).

## Prerequisites

1. The WebSocket service must be running (see main README for setup)
2. A valid JWT token for authentication

## Installation

```bash
cd cmd/client
go mod download
```

## Usage

```bash
go run main.go -token <your_jwt_token> -endpoint ws://localhost:8009/connection -ajaib-id <your_ajaib_id>
```

### Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `-token` | *required* | JWT token for authentication |
| `-endpoint` | `ws://localhost:8009/connection` | WebSocket endpoint URL |
| `-ajaib-id` | `130010505` | Ajaib user ID for channel subscription |

## Example

```bash
./cmd/client/token-gen.sh 130010505
# Ajaib ID: 130010505
# Token:    e30.eyJzdWIiOiIxMzAwMTA1MDUifQ.fake

# Use the generated token
export JWT_TOKEN="e30.eyJzdWIiOiIxMzAwMTA1MDUifQ.fake"

go run main.go \
    -token $JWT_TOKEN \
    -endpoint "ws://localhost:8009/connection" \
    -ajaib-id "130010505"
```

## What It Does

1. **Connects** to the WebSocket server using the JWT token
2. **Subscribes** to two channels:
   - `user:{ajaib_id}:margin` - Margin updates
   - `user:{ajaib_id}:position` - Position updates
3. **Prints** incoming publications to the console

## Expected Output

```
2026/02/19 14:30:24 Connecting to server... (code: 0, reason: connect called)
2026/02/19 14:30:24 Subscribed to margin channel: user:130010505:margin
2026/02/19 14:30:24 Subscribed to position channel: user:130010505:position
2026/02/19 14:30:24 [margin] Subscribing... (code: 0, reason: subscribe called)
2026/02/19 14:30:24 [position] Subscribing... (code: 0, reason: subscribe called)
2026/02/19 14:30:24 Connected to server (client_id: 88df90ff-3c11-4048-a508-2b3eafc60002)
2026/02/19 14:30:24 [margin] Subscribed successfully (recovered: false)
2026/02/19 14:30:24 [position] Subscribed successfully (recovered: false)
2026/02/19 14:43:36 [margin] {"timestamp":1771247920575,"cfx_user_id":"24223","asset":"USDT","total_position_value":0,"margin_balance":226.557992,"order_margin":0.309125,"effective_leverage":0,"maintenance_margin":0,"unrealized_pnl":0,"available_margin":226.248867,"wallet_balance":226.557992,"margin_ratio":0,"withdrawable_margin":226.248867}
2026/02/19 14:43:36 [position] {"timestamp":1771120700990,"cfx_user_id":"24223","symbol":"PEPEUSDT-PERP","size":0,"value":0,"leverage":25,"entry_price":0,"mark_price":0.0000048079,"liquidation_price":0,"maintenance_margin":0,"realised_pnl":0,"unrealised_pnl":0,"deleverage_percentile":0,"risk_limit":1000000,"open_order_buy_cost":6.713,"open_order_sell_cost":0,"initial_margin_requirement":0.04,"updated_time":1771120700504,"open_order_buy_quantity":10000000,"open_order_sell_quantity":0,"order_margin":0.309125}
...
```

Press `Ctrl+C` to disconnect and exit.
