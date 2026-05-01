The current implementation includes a RESP2 parser, command dispatching, and a
TCP server that can handle basic Redis-style commands.

# Current implementation

Main pieces:

- [`cmd/redis-server/main.go`](cmd/redis-server/main.go): program entry; TCP listen and accept loop
- [`internal/server/handler.go`](internal/server/handler.go): per-connection RESP read loop and error/reply writes
- [`internal/redis/dispatch.go`](internal/redis/dispatch.go): `DispatchCommand`, string store (`SET` / `GET`), and command handlers
- [`internal/redis/list.go`](internal/redis/list.go): list type, in-memory list map, LRANGE indexing, push/pop primitives
- [`internal/redis/dispatch_test.go`](internal/redis/dispatch_test.go): integration tests for dispatch
- [`internal/resp/resp.go`](internal/resp/resp.go): RESP2 parser, encoders, command decoding
- [`internal/resp/resp_test.go`](internal/resp/resp_test.go): parser and encoder unit tests

Supported RESP2 types:

- Simple strings: `+OK\r\n`
- Errors: `-ERR bad\r\n`
- Integers: `:42\r\n`
- Bulk strings: `$5\r\nhello\r\n`
- Arrays: `*2\r\n$4\r\nECHO\r\n$2\r\nhi\r\n`

Supported commands include (non-exhaustive): `PING`, `ECHO`, `SET`, `GET`,
`RPUSH`, `LPUSH`, `LRANGE`, `LLEN`, `LPOP`, `BLPOP` (synchronous subset), and related list operations.

# Run locally

Ensure you have Go 1.26+ installed (`go version`), then start the server:

```sh
go run ./cmd/redis-server
```

Or build a binary:

```sh
go build -o redis-server ./cmd/redis-server
./redis-server
```

The server listens on `0.0.0.0:6379`.

You can test it with `redis-cli`:

```sh
redis-cli ping
redis-cli echo "hello"
```

# Run tests

```sh
go test ./...
```

# Notes

- The parser uses `bufio.Reader` so it can handle partial reads and pipelined
  RESP messages correctly.
- Arrays are parsed recursively because each array element is itself a full RESP
  value.
- This project targets RESP2, which aligns with Codecrafters Redis stages.
