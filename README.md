# Redis Go Implementation

A multi-threaded, simple Redis-compatible server built from the ground up in Go.

## Motivation

This project started with a simple question: _How does Redis manage millions of operations with such elegant simplicity?_ My research into the Redis core repository revealed a fascinating architecture: a single-threaded system powered by an Event Loop. This inspired me to rebuild those internals in Go, transitioning from a single-threaded model to a modern, concurrent architecture leveraging Goroutines and Channels.

## Features

**RESP2 Protocol** • **Thread-safe Storage** • **Strings** • **Lists with Blocking Reads** • **Sorted Sets** • **Streams with Range Reads** • **Transactions (MULTI/EXEC/WATCH)** • **Pub/Sub Messaging** • **AOF Persistence** • **Replication** • **Custom CLI**

- **Redis Serialization Protocol**: Implemented the full **RESP2** serialization protocol with a recursive, low-allocation parser.
- **Concurrency**: Leveraged Go's concurrency primitives (channels, mutexes, and `sync.Cond`) to handle blocking operations like `BLPOP` and `XREAD`.
- **Atomic Operations**: Designed an **Optimistic Locking** mechanism using version tracking to support `WATCH` transactions.
- **Core Data Types**: Added Redis-style strings, lists, sorted sets, and streams with command coverage for common read/write workflows.
- **List Operations**: Implemented queue-like and stack-like list behavior with push/pop, range reads, and blocking pop support.
- **Stream Processing**: Implemented append-only stream entries, range scans, and blocking stream reads.
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

| Category     | Commands                                             |
| ------------ | ---------------------------------------------------- |
| Strings      | `SET`, `GET`, `INCR`                                 |
| Lists        | `LPUSH`, `RPUSH`, `LPOP`, `LRANGE`, `BLPOP`          |
| Sorted Sets  | `ZADD`, `ZRANK`, `ZRANGE`, `ZCARD`, `ZSCORE`, `ZREM` |
| Streams      | `XADD`, `XRANGE`, `XREAD`                            |
| Transactions | `MULTI`, `EXEC`, `DISCARD`, `WATCH`, `UNWATCH`       |
| Pub/Sub      | `PUBLISH`, `SUBSCRIBE`, `UNSUBSCRIBE`                |
| System       | `PING`, `ECHO`, `TYPE`, `INFO`, `CONFIG`             |

## Future Work

- [ ] LRU cache
- [ ] HyperLogLog
- [ ] Bloom filter
- [ ] Count-min sketch
- [ ] Geospatial commands
- [ ] Authentication
- [ ] Cuckoo filter
- [ ] Graceful shutdown

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
