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
