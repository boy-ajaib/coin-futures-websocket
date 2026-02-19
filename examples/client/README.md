# Centrifuge Go Client Example

This example demonstrates how to connect to the Coin Futures WebSocket service using the [Centrifuge Go SDK](https://github.com/centrifugal/centrifuge-go).

## Prerequisites

1. The WebSocket service must be running (see main README for setup)
2. A valid JWT token for authentication

## Installation

```bash
cd examples/client
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
# With a JWT token
export JWT_TOKEN="eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9..."

go run main.go \
  -token $JWT_TOKEN \
  -endpoint ws://localhost:8009/connection \
  -ajaib-id 130010505
```

## What It Does

1. **Connects** to the WebSocket server using the JWT token
2. **Subscribes** to two channels:
   - `user:{ajaib_id}:margin` - Margin updates
   - `user:{ajaib_id}:position` - Position updates
3. **Prints** incoming publications to the console

## Expected Output

```
2025/02/16 10:00:00 Connecting to server...
2025/02/16 10:00:00 Connected to server (client_id: abc123)
2025/02/16 10:00:00 [margin] Subscribing...
2025/02/16 10:00:00 [margin] Subscribed successfully (recovered: false)
2025/02/16 10:00:00 Subscribed to margin channel: user:130010505:margin
2025/02/16 10:00:00 [position] Subscribing...
2025/02/16 10:00:00 [position] Subscribed successfully (recovered: false)
2025/02/16 10:00:00 Subscribed to position channel: user:130010505:position
2025/02/16 10:00:05 [margin] Update received - CFX UserID: cfx_12345, Asset: USDT
2025/02/16 10:00:05 [margin]   Margin Balance: 1000.50
2025/02/16 10:00:05 [margin]   Wallet Balance: 1050.00
```

Press `Ctrl+C` to disconnect and exit.

## Integration in Your Application

To use this in your own Go application:

```go
import "github.com/centrifugal/centrifuge-go"

// Create client
opts := centrifuge.DefaultConfig()
opts.Token = "your_jwt_token"
client := centrifuge.New("ws://localhost:8009/connection", opts, centrifuge.JSON())

// Set up handlers
client.OnConnected(func(ctx context.Context, e centrifuge.ConnectedEvent) {
    // Connected!
})

// Subscribe to channel
sub, _ := client.NewSubscription("user:130010505:margin")
sub.OnPublication(func(ctx context.Context, e centrifuge.PublicationEvent) {
    // Handle publication
    var data MarginPublication
    json.Unmarshal(e.Data, &data)
    // Process data...
})
sub.Subscribe()

// Connect
client.Connect()
```

For more details, see the [Centrifuge Go documentation](https://github.com/centrifugal/centrifuge-go).
