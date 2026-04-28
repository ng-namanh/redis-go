The current implementation includes a RESP2 parser, command dispatching, and a
TCP server that can handle basic commands like `PING` and `ECHO`.

# Current implementation

Main pieces:

- `app/main.go`: TCP server, connection loop, RESP read/write flow
- `app/dispatch.go`: command dispatch for `PING` and `ECHO`
- `internal/resp/resp.go`: RESP2 parser, encoders, and command decoding
- `internal/resp/resp_test.go`: parser and encoder tests

Supported RESP2 types:

- Simple strings: `+OK\r\n`
- Errors: `-ERR bad\r\n`
- Integers: `:42\r\n`
- Bulk strings: `$5\r\nhello\r\n`
- Arrays: `*2\r\n$4\r\nECHO\r\n$2\r\nhi\r\n`

Supported commands:

- `PING` -> `+PONG\r\n`
- `ECHO <message>` -> bulk string reply

# Run locally

Ensure you have `go 1.26` installed, then start the server:

```sh
./your_program.sh
```

The server listens on `127.0.0.1:6379` (via `0.0.0.0:6379` in code).

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
- This project currently targets RESP2, which is what the Codecrafters Redis
  stages use.
