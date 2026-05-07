# Redis Go Implementation

A lightweight Redis-compatible server implemented in Go, targeting RESP2 protocol compatibility.

## Current Implementation

The project is organized into several internal packages to separate concerns:

- [**`cmd/redis-server/main.go`**](cmd/redis-server/main.go): Program entry point; sets up the TCP listener and accept loop.
- [**`internal/server/handler.go`**](internal/server/handler.go): Manages the lifecycle of TCP connections.
- [**`internal/client/`**](internal/client/): Manages per-connection state, including transaction queues (`MULTI`, `EXEC`, `DISCARD`).
- [**`internal/commands/`**](internal/commands/): Contains the core logic for all supported Redis commands:
    - `string.go`: String operations (`SET`, `GET`, `INCR`).
    - `list.go` & `list_blocking.go`: List operations (`LPUSH`, `RPUSH`, `LPOP`, `BLPOP`, etc.).
    - `stream.go` & `xread.go`: Stream operations (`XADD`, `XRANGE`, `XREAD`).
    - `store.go`: Central in-memory data store with thread-safe access.
- [**`internal/resp/`**](internal/resp/): Robust RESP2 parser and encoder.
- [**`internal/redis/`**](internal/redis/): A compatibility shim and test suite for the server logic.

## Supported Commands

The server currently supports a wide range of Redis commands:

- **General**: `PING`, `ECHO`, `TYPE`
- **Strings**: `SET` (with `PX` expiry), `GET`, `INCR`
- **Lists**: `LPUSH`, `RPUSH`, `LPOP`, `LLEN`, `LRANGE`, `BLPOP`
- **Streams**: `XADD`, `XRANGE`, `XREAD` (with `BLOCK` and `COUNT` support)
- **Transactions**: `MULTI`, `EXEC`, `DISCARD`

## Features

- **Stateful Sessions**: Each connection has its own client state, enabling atomic transactions.
- **Blocking Operations**: Support for blocking commands like `BLPOP` and `XREAD BLOCK`.
- **RESP2 Protocol**: Full support for Simple Strings, Errors, Integers, Bulk Strings, and Arrays.
- **Thread Safety**: Global mutex-protected store ensures data integrity across concurrent connections.

## Getting Started

### Run locally

Ensure you have Go 1.26+ installed, then start the server:

```sh
go run ./cmd/redis-server
```

The server listens on `0.0.0.0:6379`. You can interact with it using `redis-cli`:

```sh
redis-cli SET foo bar
redis-cli GET foo
```

### Run tests

The project includes a comprehensive test suite covering protocol parsing, command logic, and transaction isolation.

```sh
go test ./...
```

## Technical Notes

- **Buffered I/O**: The parser uses `bufio.Reader` to handle network packets efficiently and support pipelining.
- **Recursive Parsing**: RESP Arrays are parsed recursively, allowing for complex nested structures.
- **Atomic Transactions**: Commands queued within a `MULTI` block are executed sequentially when `EXEC` is called, with results returned as a single array.
