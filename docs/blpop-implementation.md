# BLPOP (blocking list pop) — deep implementation notes

This document explains how **blocking `BLPOP` with FIFO ordering** is implemented in this repo, and how it interacts with list operations (`LPUSH`/`RPUSH`/`LPOP`) and RESP encoding.

Relevant files:

- `internal/redis/blocking.go`
- `internal/redis/list.go`
- `internal/redis/dispatch.go`

---

## English

### What we are implementing (behavioral target)

`BLPOP key [key ...] timeout`:

- If any provided key has a **non-empty list**, pop **one element from the left** from the *first non-empty key* (scan keys left→right) and return an array: `[key, value]`.
- If **all** lists are empty (or missing), **block** until either:
  - An element becomes available (due to `LPUSH` or `RPUSH` on one of the keys), then pop+return `[key, value]`, or
  - Timeout expires → return **null array** in this codebase as `*-1\r\n`.
- If any key is present but holds the wrong type (in our simplified model: exists in `cache` as a string), return **WRONGTYPE**.

This repo is not implementing full Redis object types; it uses **two stores**:

- `cache map[string]any` for string values (`SET`/`GET`)
- `lists map[string]list` for list values (where `type list []string`)

So “wrong type” is modeled as: **the key exists in `cache`** when a list command expects a list.

---

### Data structures

#### List store

In `internal/redis/list.go`:

- `type list []string`
- `var lists = make(map[string]list)`

Core helpers:

- `listsRPush(key, elems)` appends to the tail.
- `listsLPush(key, vals)` prepends (while preserving Redis ordering semantics).
- `getPoppedElements(key, n)` removes up to `n` elements from the head.
  - It also deletes `lists[key]` when the list becomes empty to keep “missing key = empty list” semantics.

#### Blocking waiter queue (FIFO across clients)

In `internal/redis/blocking.go`:

```go
type blpopWaiter struct {
  keys []string
  ch   chan []byte // cap=1; carries the encoded RESP reply
}

var blpopWaiters []*blpopWaiter
```

Each blocked `BLPOP` call creates a `blpopWaiter` and appends it to `blpopWaiters`. This list is the **global FIFO** ordering across clients.

---

### Concurrency: what `listMu` protects

Both list mutations and waiter queue operations use the same mutex:

- `var listMu sync.Mutex` (declared in `blocking.go`, used in `list.go` too)

That lock protects **all** of:

- the `lists` map and the underlying slice contents
- the `blpopWaiters` queue
- the “serve a waiter by popping an element” critical section (so a value cannot be delivered twice)

Important: `SET/GET` use `cache` without `listMu`. This is a simplified model; list commands do check `cache` for WRONGTYPE, but not under a broader “global keyspace” lock.

---

### The fast path: try to pop immediately

`BLPOP` begins under lock by checking WRONGTYPE, then trying to pop immediately:

- `blpopTryPop(keys)` loops keys left→right and calls `getPoppedElements(k, 1)`.
- If any key returns 1 popped element, `BLPOP` immediately returns the RESP array `[key, value]`.

This matches the idea that `BLPOP` is only blocking when it has to.

---

### The blocking path: enqueue + wait

If nothing is available, `BLPOP`:

1. Creates a buffered channel `ch := make(chan []byte, 1)`.
2. Copies keys to detach from any caller slice:
  - `keysCopy := append([]string(nil), keys...)`
3. Appends waiter to the global queue:
  - `blpopWaiters = append(blpopWaiters, w)`
4. Releases `listMu`.
5. Waits:
  - If timeout t > 0: wait on `reply := <-ch` OR timer.
  - If timeout t == 0: wait forever on `reply := <-ch`.

#### Why buffered channel of size 1?

It avoids the “waker goroutine blocks forever” issue: the waker will do a non-blocking send into a 1-capacity channel. In normal flow, the receiver is actively waiting, so the send should succeed immediately anyway.

---

### Waking blocked clients: `flushBLPOPAfterPush`

After every `LPUSH` and `RPUSH`, we do:

```go
flushBLPOPAfterPush()
```

This function repeatedly tries to serve waiters until no more can be served with the current list state:

- `tryServeOneBLPOPWaiter()` scans the waiter queue in order and finds the earliest waiter that can be satisfied.

#### How a waiter is “satisfied”

Inside `tryServeOneBLPOPWaiter()` (still under `listMu`):

1. For a waiter `w`, check which watched key is currently non-empty:
  - `firstNonEmptyListKey(w.keys)` scans `w.keys` left→right and checks `len(lists[k]) > 0`.
2. If some key `k` is non-empty, pop exactly one:
  - `popped := getPoppedElements(k, 1)`
3. Remove this waiter from `blpopWaiters` queue (FIFO):
  - `blpopWaiters = append(blpopWaiters[:i], blpopWaiters[i+1:]...)`
4. Encode reply bytes as RESP array `[key, value]` and send to `w.ch`.

Because removal happens before send and the whole operation is under the lock, we guarantee:

- That popped element is not reused.
- That waiter is only served once.

#### “Skip” behavior is intentional (multi-key BLPOP)

If the earliest waiter is waiting on keys that are still empty, we **skip** it and try later waiters. This matters when:

- Waiter A: `BLPOP k1 0`
- Waiter B: `BLPOP k2 0`
- Someone pushes to `k2`

If we only checked the queue head, B would never be served until k1 becomes non-empty. Skipping allows the push to `k2` to unblock B immediately while preserving FIFO *among waiters that are actually satisfiable* at that moment.

---

### Timeout cleanup

When timeout fires, we must remove the waiter from the queue so it doesn’t get served later:

```go
listMu.Lock()
removeWaiterIfQueued(w)
listMu.Unlock()
return []byte("*-1\r\n"), nil
```

This is why the test helper `resetStores()` clears `blpopWaiters` under `listMu`: otherwise a timed-out waiter could remain, or a blocking waiter from one test could affect the next.

---

### RESP reply encoding for BLPOP

`BLPOP` returns an **array of two bulk strings**:

- key as bulk string
- value as bulk string

Encoded by `encodeBLPOPReply()`:

```go
resp.WriteArray([]resp.RESP{
  {Type: resp.BulkString, Str: key},
  {Type: resp.BulkString, Str: val},
})
```

Timeout returns `*-1\r\n` (null array in this codebase).

---

### Walkthrough scenarios (timelines)

#### Scenario A: fast path (no blocking)

- Initial: `lists["a"] = ["x"]`
- Client: `BLPOP a b 5`
- Under lock:
  - `a` has element → pop `"x"`
  - Return `["a", "x"]`
- No waiter is enqueued.

#### Scenario B: block then wake via RPUSH

- Initial: `lists` empty
- Client 1: `BLPOP a 5`
  - empty → enqueue waiter W1(keys=["a"])
  - wait on W1.ch or timer
- Client 2: `RPUSH a v1`
  - under lock: append `"v1"`
  - `flushBLPOPAfterPush()` finds W1 satisfiable
  - pops `"v1"`, removes W1 from queue, sends reply bytes
- Client 1 receives reply and returns `["a","v1"]`.

#### Scenario C: global FIFO across multiple clients

Suppose two clients wait on the same key:

- Client 1: `BLPOP a 0` → waiter W1 enqueued first
- Client 2: `BLPOP a 0` → waiter W2 enqueued second
- Producer: `RPUSH a v1 v2` (or two pushes)

`flushBLPOPAfterPush()` will:

- Serve W1 first, popping `v1`
- Then loop again and serve W2, popping `v2`

So clients are unblocked in FIFO order.

---

### Known limitations (explicitly in this repo)

- `timeout == 0` waits forever and is **not cancelled** on connection close (no context propagation yet).
- `cache` vs `lists` is not guarded by a single global mutex; we only coordinate list + BLPOP waiters with `listMu`.
- This is a minimal subset; it’s not a full Redis type system.

---

## Tiếng Việt

### Mục tiêu hành vi (Redis-like)

`BLPOP key [key ...] timeout`:

- Nếu có key nào có list **không rỗng**, pop **1 phần tử bên trái** từ *key đầu tiên không rỗng* (duyệt trái→phải) và trả về mảng: `[key, value]`.
- Nếu tất cả list rỗng/không tồn tại thì **chặn** cho tới khi:
  - Có phần tử mới (do `LPUSH`/`RPUSH` vào một key đang chờ) → pop và trả `[key, value]`, hoặc
  - Hết timeout → trả **null array** (ở repo này là `*-1\r\n`).
- Nếu key “sai kiểu” (trong mô hình đơn giản của repo: key đang tồn tại trong `cache` như string) → trả **WRONGTYPE**.

Repo dùng 2 store:

- `cache map[string]any` cho string (`SET`/`GET`)
- `lists map[string]list` cho list (`type list []string`)

---

### Cấu trúc dữ liệu

#### Store cho list

Trong `internal/redis/list.go`:

- `lists` là map từ key → `[]string`.
- `RPUSH` append vào cuối.
- `LPUSH` prepend (đúng thứ tự như Redis).
- `getPoppedElements(key, n)` lấy từ đầu và **xóa key khỏi map** khi list rỗng (để “key không tồn tại” tương đương “list rỗng”).

#### Hàng đợi waiter cho BLPOP (FIFO toàn cục)

Trong `internal/redis/blocking.go`:

- Mỗi `BLPOP` bị block tạo một `blpopWaiter{keys, ch}` rồi append vào `blpopWaiters`.
- `blpopWaiters` chính là FIFO giữa các client.

---

### Đồng bộ: `listMu` bảo vệ những gì

`listMu` bảo vệ toàn bộ vùng dữ liệu liên quan đến list và BLPOP:

- `lists`
- `blpopWaiters`
- thao tác “pop để phục vụ một waiter” (tránh pop trùng)

`SET/GET` ghi/đọc `cache` không dùng `listMu` (mô hình đơn giản).

---

### BLPOP: đường nhanh (fast path)

Trong lock:

- kiểm tra WRONGTYPE (nếu key nằm trong `cache`)
- thử pop ngay theo thứ tự keys trái→phải
- nếu pop được thì trả `[key, value]` và không enqueue waiter

---

### BLPOP: đường chặn (enqueue + wait)

Nếu chưa pop được:

- tạo `ch` buffered size 1
- copy `keys` để tránh alias slice
- enqueue vào `blpopWaiters`
- unlock rồi chờ:
  - timeout > 0: chờ reply hoặc timer
  - timeout == 0: chờ vô hạn (chưa có cancel khi client disconnect)

---

### Wake-up: `flushBLPOPAfterPush` sau LPUSH/RPUSH

Sau mỗi `LPUSH`/`RPUSH`, gọi `flushBLPOPAfterPush()`:

- nó lặp `tryServeOneBLPOPWaiter()` cho tới khi không còn waiter nào có thể phục vụ với trạng thái list hiện tại

Điểm quan trọng:

- nếu waiter đầu queue chưa có key nào non-empty, ta **bỏ qua** nó để tìm waiter sau có thể phục vụ (hữu ích cho trường hợp mỗi waiter chờ key khác nhau).

---

### Dọn waiter khi timeout

Khi timeout:

- lock lại
- remove waiter khỏi `blpopWaiters`
- trả `*-1\r\n`

Vì vậy `resetStores()` trong test phải xóa `blpopWaiters` dưới lock để tránh leak giữa các test.

---

### Giới hạn hiện tại

- `timeout == 0` chờ vô hạn và chưa hủy khi đóng kết nối.
- `cache` và `lists` chưa được bảo vệ bởi một mutex chung.
- Chỉ là subset tối thiểu, chưa phải Redis đầy đủ.

