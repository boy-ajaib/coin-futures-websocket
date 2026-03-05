# Coin Futures Websocket

A Go-based WebSocket service for connecting to the CFX (Coin Futures Exchange) using the Centrifuge protocol.

## Usage

```bash
$ make

Usage:
  make <target>

Targets:
  help              Show this help message
  run               Run the server using default config
  run.dev           Run the server using development config
  test              Run the tests of the project
  test.race         Run the tests of the project while also checking race conditions
  test.verbose      Run the tests of the project while also checking race conditions (verbose)
  test.coverage     Run the tests of the project and export the coverage
  fmt               Format '*.go' files with gofumpt
  lint              Run linter using golangci-lint
  lint.fix          Run linter using golangci-lint and fix it
  build             Build the server
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
# copy the default config to development config
cp ./config/config.yml ./config/development.yml

# make some changes to the development config

# run the server with development config
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

### Centrifuge Client SDKs

Centrifuge uses its own binary protocol over WebSocket, so raw WebSocket clients (like Postman) won't work. Use the official Centrifuge client SDKs:

**Go Client:**
```bash
go get github.com/centrifugal/centrifuge-go
```

**JavaScript Client:**
```bash
npm install centrifuge
```

See the [Centrifuge documentation](https://centrifugal.dev/) for more details.

### Testing via an example client

We have prepared an example client in `cmd/client` folder. Please follow the instructions in `cmd/client/README.md` to test the server.

To summarize:
1. Generate a JWT token for the user via `./cmd/client/token-gen.sh <ajaib_id>`
2. Run the client via `go run cmd/client/main.go -token <jwt_token> -endpoint ws://localhost:8009/connection -ajaib-id <ajaib_id>`
3. Check the redis broker via `redis-cli MONITOR | grep "coin-futures-websocket-dev"`

### Testing via Postman

Centrifuge cannot be tested via Postman. Please use the Centrifuge client SDKs to test the server.

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

Please remember the port since we will need it below.

### Run coin-futures-websocket

Create a new file in `config` folder called `development.yml`. The example content might be changed, please refer to `config/config.yml` for the default config.

```yaml
app:
    env: development
    log_level: debug

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
    max_message_age_ms: 5000

websocket_server:
    enabled: true
    port: 8009
    ping_interval_ms: 2000
    ping_timeout_ms: 30000
    max_connections_per_user: 5
    shutdown_timeout_ms: 10000

centrifuge:
    node_name: coin-futures-websocket-dev
    namespace: coin-futures-websocket-dev
    log_level: debug
    client_insecure: true
    client_announce: true
    presence: false
    join_leave: false
    history_size: 0
    history_ttl_seconds: 0
    force_recovery: false
    redis_broker:
        enabled: true
        address: "127.0.0.1:6379"
        password: ""
        db: 0
        prefix: "coin-futures-websocket-dev"
        connect_timeout_ms: 1000
        io_timeout_ms: 4000

coin_cfx_adapter:
    host: http://localhost:8889
    cache_ttl_seconds: 60

coin_data:
    host: http://coin-data-svc.stg.ajaib.int
    cache_ttl_seconds: 60
    cfx_usdt_asset: "USDT"

coin_setting:
    host: http://coin-setting-svc.stg.ajaib.int
```

Now, we run `coin-futures-websocket`.

```sh
make run.dev
# or
ENV=development go run cmd/server/main.go
```

Running with above command will automatically use `config/development.yml` as our config file.
