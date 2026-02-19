# Coin Futures Websocket

A Go-based WebSocket service for connecting to the CFX (Coin Futures Exchange) using the Centrifuge protocol.

## Usage

```bash
$ make

Usage:
  make <target>

Targets:
  help              Show this help message
  run               Run the app using default config
  run.dev           Run the app using development config
  test              Run the tests of the project
  test.verbose      Run the tests of the project (verbose)
  test.coverage     Run the tests of the project and export the coverage
  fmt               Format '*.go' files with gofumpt
  build             Build the app
```

## Building

```bash
$ make build
```

## Running

```bash
$ make run
```

## Development

Run with development config:

```bash
make run.dev
```

## WebSocket Protocol

This service uses the **Centrifuge protocol** for real-time WebSocket communication. Centrifuge is a production-grade messaging protocol with built-in support for:

- Automatic reconnection with exponential backoff
- Binary and JSON message formats
- Channel-based subscriptions
- Server-side push notifications

### Connection Endpoint

```
ws://localhost:8009/connection
```

### Authentication

Authentication is done via JWT token. The server supports two methods:

1. **Centrifuge Token Flow** (recommended): Include the token in the connect command
2. **HTTP Header Flow**: Include the token in `X-Socket-Authorization` header

The JWT payload must contain the Ajaib user ID in the `sub` claim.

### Channel Format

Channels follow this naming convention:

```
user:{ajaib_id}:{type}
```

Where `type` can be:
- `margin` - User margin updates
- `position` - User position updates

Example channels:
- `user:130010505:margin`
- `user:130010505:position`

### Centrifuge Protocol Examples

#### Connect Command

```json
{
  "id": 1,
  "subscribe": {
    "user:130010505:margin": {}
  },
  "token": "eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature"
}
```

#### Subscribe Command (after connection)

```json
{
  "id": 2,
  "subscribe": {
    "user:130010505:margin": {}
  }
}
```

```json
{
  "id": 3,
  "subscribe": {
    "user:130010505:position": {}
  }
}
```

#### Server Push (Publication)

When a margin update occurs, the server pushes:

```json
{
  "push": {
    "channel": "user:130010505:margin",
    "pub": {
      "cfx_user_id": "cfx_12345",
      "asset": "USDT",
      "margin_balance": "1000.50",
      "updated_at": 1704067200000
    }
  }
}
```

### Using Centrifuge Client SDKs

We recommend using official Centrifuge client SDKs instead of raw WebSocket:

**Go Client:**
```bash
go get github.com/centrifugal/centrifuge-go
```

**JavaScript Client:**
```bash
npm install centrifuge
```

See the [Centrifuge documentation](https://centrifugal.dev/) for more details.

### Testing via Postman

To test manually via Postman:

1. Create a new WebSocket request
2. URL: `ws://localhost:8009/connection`
3. Headers:
   - `X-Socket-Authorization: Bearer <your_jwt_token>`
4. Click "Connect"
5. After connection, send subscribe commands using Centrifuge protocol format

**Important Notes:**
- Users can only subscribe to their own channels (validated by ajaib_id in JWT)
- The connection endpoint is `/connection` (not `/ws`)
- Messages use Centrifuge protocol format, not custom JSON

## Database Schemas

### User Margin Snapshot

| Nullable? | Column Name | Data Type | Description |
|-----------|-------------|-----------|-------------|
| null      | id          | binary(16) | Unique identifier |
| not null  | cfx_user_id | varchar(30) | User ID |
| not null  | asset       | varchar(100) | Asset |
| null      | margin_balance | decimal(48, 18) | Margin balance |
| null      | order_margin | decimal(48, 18) | Order margin |
| null      | unrealized_pnl | decimal(48, 18) | Unrealized PnL |
| null      | wallet_balance | decimal(48, 18) | Wallet balance |
| null      | margin_ratio | decimal(48, 18) | Margin ratio |
| null      | withdrawable_margin | decimal(48, 18) | Withdrawable margin |
| not null  | created_at | timestamp(3) | Creation timestamp |
| not null  | updated_at | timestamp(3) | Update timestamp |
| null      | deleted_at | timestamp(3) | Deletion timestamp |

### User Position Snapshot

| Nullable? | Column Name | Data Type | Description |
|-----------|-------------|-----------|-------------|
| not null  | id | binary(16) | Unique identifier |
| not null  | cfx_user_id | varchar(30) | User ID |
| not null  | symbol | varchar(20) | Symbol |
| null      | size | decimal(48, 18) | Position size |
| null      | leverage | decimal(48, 18) | Leverage |
| null      | entry_price | decimal(48, 18) | Entry price |
| null      | mark_price | decimal(48, 18) | Mark price |
| null      | liquidation_price | decimal(48, 18) | Liquidation price |
| null      | unrealised_pnl | decimal(48, 18) | Unrealized PnL |
| not null  | created_at | timestamp(3) | Creation timestamp |
| not null  | updated_at | timestamp(3) | Update timestamp |
| null      | deleted_at | timestamp(3) | Deletion timestamp |


## How to Test

`coin-futures-websocket` depends on `coin-cfx-adapter` to get ajaib_id mapping and `coin-data` to get exchange rate. Make sure `coin-cfx-adapter` is in `feature/CRP-97` branch since the required endpoint has not been merged yet.

### Run coin-cfx-adapter

```sh
mvn clean install -DskipTests && mvn spring-boot:run -pl ap
```

Please remember the port since we will need in below.

### Run coin-futures-websocket

Create a new file in `config` folder called `development.yml`.

```yaml
app:
    env: development
    log_level: info

kafka:
    brokers:
        - kafka-1.platform.stg.ajaib.int:9092
        - kafka-2.platform.stg.ajaib.int:9092
        - kafka-3.platform.stg.ajaib.int:9092
    topics:
        - com.ajaib.coin.cfx.streamer.futures.message.UserMargin
        - com.ajaib.coin.cfx.streamer.futures.message.UserPosition
    consumer_group: coin-futures-websocket
    initial_offset: latest
    session_timeout: 10000
    heartbeat_interval: 1000

websocket_server:
    enabled: true
    port: 8009
    ping_interval_ms: 2000
    ping_timeout_ms: 30000
    max_connections_per_user: 5
    shutdown_timeout_ms: 10000

coin_cfx_adapter:
    host: http://localhost:8889
    cache_ttl_seconds: 60

coin_data:
    host: http://coin-data-svc.stg.ajaib.int
    cache_ttl_seconds: 5
    cfx_usdt_asset: "TEST"
```

Now, we run `coin-futures-websocket`.

```sh
make run.dev
# or
ENV=development go run cmd/app/main.go
```

Running with above command will automatically use `config/development.yml` as our config file.
