# Redis Go Implementation

A multi-threaded, simple Redis-compatible server built from the ground up in Go.

## Motivation
This project started with a simple question: *How does Redis manage millions of operations with such elegant simplicity?* My research into the Redis core repository revealed a fascinating architecture: a single-threaded system powered by an Event Loop. This inspired me to rebuild those internals in Go, transitioning from a single-threaded model to a modern, concurrent architecture leveraging Goroutines and Channels.

## Features
**RESP2 Protocol** • **Thread-safe Storage** • **Transactions (MULTI/EXEC/WATCH)** • **Sorted Sets** • **Streams** • **AOF Persistence** • **Replication** • **Custom CLI**

## What this project covers so far:
- **Redis Serialization Protocol**: Implemented the full **RESP2** serialization protocol with a recursive, low-allocation parser.
- **Concurrency**: Leveraged Go's concurrency primitives (channels, mutexes, and `sync.Cond`) to handle blocking operations like `BLPOP` and `XREAD`.
- **Atomic Operations**: Designed an **Optimistic Locking** mechanism using version tracking to support `WATCH` transactions.
- **Redis AOF**: Developed an **Append-Only File (AOF)** engine with manifest management for reliable state recovery.
- **Messaging Architecture**: Built a **Pub/Sub** engine using thread-safe subscriber mapping and asynchronous message broadcasting.

## Knowledge Gained
Building this project was a deep dive into Redis internals, helping me to understand multi-threading models and concurrency control mechanisms.
### Redis & Database Internals
- **Serialization**: Deep understanding of the **RESP** protocol and efficient data serialization over TCP.
- **In-memory Data Structures**: Implementing the logic behind **Sorted Sets**, **Streams**, and **Lists**.
- **Transaction Mechanics**: Understanding atomicity, `MULTI/EXEC` flow, and **Optimistic Locking** with `WATCH`.
- **Persistence and Recovery**: How the **Append-Only File (AOF)** ensures data durability and recovery.
- **Distributed Basics**: Implementing **Master-Slave replication** handshakes and command propagation.

### Advanced Go Development
- **Concurrency**: Advanced use of `sync.Mutex`, `sync.RWMutex`, and `sync.Cond` for thread-safe data access.
- **Asynchronous Coordination**: Using **Channels** to manage blocking operations (`BLPOP`, `XREAD`) and Pub/Sub broadcasting.
- **Networking**: Handling raw TCP connections and stream parsing with the `net` and `io` packages.

## Supported Commands
- **Strings**: `SET`, `GET`, `INCR`
- **Lists**: `LPUSH`, `RPUSH`, `LPOP`, `LRANGE`, `BLPOP`
- **Sorted Sets**: `ZADD`, `ZRANK`, `ZRANGE`, `ZCARD`, `ZSCORE`, `ZREM`
- **Streams**: `XADD`, `XRANGE`, `XREAD`
- **Transactions**: `MULTI`, `EXEC`, `DISCARD`, `WATCH`, `UNWATCH`
- **Pub/Sub**: `PUBLISH`, `SUBSCRIBE`, `UNSUBSCRIBE`
- **System**: `PING`, `ECHO`, `TYPE`, `INFO`, `CONFIG`

## Getting Started

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