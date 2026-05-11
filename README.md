# Redis Go Implementation

A lightweight, educational Redis-compatible server implemented from scratch in Go.

## 🚀 Core Features

Buildi this Redis implementation was a step-by-step exploration of distributed systems, protocol design, and efficient data structures:

1.  **The Foundation (RESP2 Protocol)**: The journey began with implementing the REdis Serialization Protocol. This involved building a robust, recursive parser capable of handling Simple Strings, Errors, Integers, Bulk Strings, and nested Arrays.
2.  **Core Storage & Strings**: Implementing a thread-safe global map to store values and the basic `GET`, `SET`, and `INCR` commands.
3.  **Concurrency & Atomic Transactions**: Moving beyond simple commands to support `MULTI`, `EXEC`, and `WATCH`. This required managing per-connection state and implementing version-based optimistic locking for keys.
4.  **Advanced Data Structures**:
    - **Lists**: Implementing doubly-linked lists with blocking operations (`BLPOP`) using Go channels for coordination.
    - **Streams**: Building an append-only log structure with support for ID-based range queries (`XRANGE`) and blocking reads (`XREAD`).
    - **Sorted Sets (Skip Lists)**: Implementing a probabilistic **Skip List** with `span` tracking, allowing for $O(\log N)$ rank calculations—a core optimization used by the original Redis.
5.  **Persistence (AOF)**: Ensuring data survives restarts by implementing an Append-Only File with manifest management and a background fsync policy.
6.  **Replication**: Establishing basic master-slave handshakes and command propagation to support horizontal read scaling.
7.  **Custom CLI**: Completing the ecosystem by building a Go-based CLI client that supports interactive REPL mode, quoted arguments, and formatted output.

## 🏗️ Project Architecture

The codebase is organized into modular internal packages:

- [**`cmd/`**](cmd/): Entry points for the `redis-server` and `redis-cli`.
- [**`internal/resp/`**](internal/resp/): The heartbeat of the system—a dedicated RESP2 parser and encoder.
- [**`internal/commands/`**](internal/commands/): Logic for Strings, Lists, Streams, Sorted Sets, and Pub/Sub.
- [**`internal/store/`**](internal/commands/store.go): The thread-safe in-memory engine.
- [**`internal/client/`**](internal/client/): Connection management and transaction state.

## 🛠️ Supported Commands

- **Strings**: `SET`, `GET`, `INCR`
- **Lists**: `LPUSH`, `RPUSH`, `LPOP`, `LLEN`, `LRANGE`, `BLPOP`
- **Sorted Sets**: `ZADD`, `ZRANK`, `ZRANGE`, `ZCARD`, `ZSCORE`, `ZREM`
- **Streams**: `XADD`, `XRANGE`, `XREAD`
- **Transactions**: `MULTI`, `EXEC`, `DISCARD`, `WATCH`, `UNWATCH`
- **Pub/Sub**: `PUBLISH`, `SUBSCRIBE`, `UNSUBSCRIBE`
- **System**: `PING`, `ECHO`, `TYPE`, `INFO`, `CONFIG`

## 🚦 Getting Started

### 1. Build & Run the Server

```sh
go build -o redis-server ./cmd/redis-server/main.go
./redis-server --port 6379
```

### 2. Connect with the CLI

```sh
go build -o redis-cli ./cmd/redis-cli/main.go
./redis-cli
```

### 3. Run the Test Suite

```sh
go test ./...
```

## 📝 Technical Highlights

- **Skip List Spans**: Unlike standard skip lists, this implementation tracks `span` (the distance between nodes) to allow $O(\log N)$ rank-based access.
- **Optimistic Locking**: `WATCH` uses a versioning system to detect concurrent modifications during a transaction.
- **Blocking Coordination**: Commands like `BLPOP` use a combination of mutexes and signaling channels to wake up clients when data becomes available.
- **AOF Manifests**: Supports incremental AOF files and replay on startup to recover state.
