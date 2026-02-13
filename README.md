# Coin Futures Websocket

A Go-based WebSocket client for connecting to the CFX (Coin Futures Exchange) using the Centrifuge protocol.

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

```
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
    no_command_timeout_ms: 20000
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

### Connect and subscribe via Postman

To connect and create a subscription via Postman, we can do this:

- Open the Postman, create new websocket request with this detail:
  - URL: ws://localhost:8009/ws
  - Header:
    - X-Socket-Authorization: Bearer eyJ0eXAiOiJBQ0NFU1MiLCJhbGciOiJFUzI1NiJ9.eyJzdWIiOiIxMzAwMTA1MDUiLCJleHAiOjE3NzA4MjA0MDAsImlhdCI6MTc3MDgxOTUwMH0.dummy_signature
- Click "Connect"
- When connected, we can subscribe to the channel by sending this message one by one:
  - {"type":"subscribe","channel":"user:130010505:margin","id":"sub-1"}
  - {"type":"subscribe","channel":"user:130010505:position","id":"sub-2"}

NOTES:
- The header name is using the same name as what our FE used.
- The JWT is just a normal JWT token. When parsed, we can see ajaib_id stored inside it as "sub". We only parsed it and used the ajaib_id to make sure the ajaib_id in the subscription payload has the same value to avoid user to subscribe to different ajaib_id.
