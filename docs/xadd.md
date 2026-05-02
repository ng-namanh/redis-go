# XADD and Redis streams

## Streams and entries

A Redis stream stores a sequence of entries in chronological order at a given key. Each entry has a **unique ID** and one or more field–value pairs.

Example:

```text
entries:
  - id: 1526985054069-0
    temperature: 36
    humidity: 95
  - id: 1526985054079-0
    temperature: 37
    humidity: 94
```

## XADD

`XADD` appends an entry. If the stream key does not exist, it is created.

Syntax (this challenge):

```text
XADD <stream_key> <id> <field1> <value1> [<field2> <value2> ...]
```

Return value: the new entry’s ID as a **bulk string**.

Optional flags (`MAXLEN`, …) are out of scope here.

---

## Auto-generated sequence only (`<ms>-*`)

The client fixes the **millisecond** part and lets the server pick the **sequence**:

- **Empty stream:** `ms-0` if `ms > 0` (e.g. `1-*` → `1-0`). If `ms == 0`, the first ID must still be **greater than `0-0`**, so this implementation uses **`0-1`** for `0-*` on an empty stream.
- **Non-empty stream:** Let the last ID be `(lastMs, lastSeq)`.
  - If `ms < lastMs` → error: ID not greater than the top item.
  - If `ms > lastMs` → new ID is `ms-0`.
  - If `ms == lastMs` → new ID is `ms-(lastSeq+1)`.

Examples:

```text
XADD some_key 1-* foo bar   → "1-0"
XADD some_key 1-* bar baz   → "1-1"
```

---

## Auto-generated full ID (`*`)

If `<id>` is `*`, the server assigns the ID:

- Use **current Unix time in milliseconds** as the time part and **0** as the sequence, when that yields an ID **strictly greater** than the stream’s last entry (including an **empty** stream).
- If the **last entry’s time part equals** the current millisecond, set the sequence to **last sequence + 1**.
- If the **system clock is behind** the last entry’s ID (last time &gt; now), bump using the last entry’s millisecond and **last sequence + 1** so the new ID stays strictly increasing (Redis-compatible).

Example:

```text
XADD stream_key * foo bar
→ bulk string e.g. "1526919030474-0"
```

---

## Entry IDs (explicit)

Each ID is two non-negative integers: `<millisecondsTime>-<sequenceNumber>` (one `-` separator).

IDs are **unique** in a stream and **strictly increasing**: every new ID must be **greater** than the last entry’s ID.

Comparison rules:

- The new ID is greater than the last if either:
  - `millisecondsTime` is **greater** than the last entry’s `millisecondsTime`, or
  - `millisecondsTime` is **equal** and `sequenceNumber` is **greater** than the last entry’s `sequenceNumber`.
- For an **empty** stream, the new ID must be **strictly greater than `0-0`**. The smallest ID Redis accepts is **`0-1`**. So **`0-0` is always invalid**.

Partial auto (`<ms>-*`) and full auto (`*`) are described above. The error messages below apply to **explicit** IDs and to **partial** IDs when `ms` is too small (not to successful `*` / `ms-*` generation).

---

## Errors (RESP simple errors)

Responses use a **simple error** (prefix `-`), for example:

- When the new ID is not strictly greater than the current top entry (including a duplicate ID), or the time part goes backwards:

  ```text
  -ERR The ID specified in XADD is equal or smaller than the target stream top item\r\n
  ```

- When the ID is **`0-0`** (always invalid):

  ```text
  -ERR The ID specified in XADD must be greater than 0-0\r\n
  ```

### Examples

```text
redis-cli XADD some_key 1-1 foo bar
"1-1"
redis-cli XADD some_key 1-1 bar baz
(error) ERR The ID specified in XADD is equal or smaller than the target stream top item
```

Second command fails: `1-1` is not strictly greater than the last ID `1-1`.

```text
redis-cli XADD some_key 1-1 foo bar
"1-1"
redis-cli XADD some_key 0-2 bar baz
(error) ERR The ID specified in XADD is equal or smaller than the target stream top item
```

`0-2` is invalid because `millisecondsTime` `0` is less than the last entry’s `1`.

```text
redis-cli XADD some_key 0-0 bar baz
(error) ERR The ID specified in XADD must be greater than 0-0
```

---

## Tester-style flow

Valid sequence:

```text
XADD stream_key 1-1 foo bar   -> bulk string "1-1"
XADD stream_key 1-2 bar baz   -> bulk string "1-2"
```

Invalid (same as last ID `1-2`):

```text
XADD stream_key 1-2 baz foo
-> -ERR The ID specified in XADD is equal or smaller than the target stream top item\r\n
```

Invalid (time smaller than top, even with larger sequence):

```text
XADD stream_key 0-3 baz foo
-> -ERR The ID specified in XADD is equal or smaller than the target stream top item\r\n
```

Invalid ID `0-0`:

```text
XADD stream_key 0-0 baz foo
-> -ERR The ID specified in XADD must be greater than 0-0\r\n
```

---

## TYPE

After a successful `XADD`, `TYPE <stream_key>` should return the simple string **`stream`** (`+stream\r\n`).

You must still return **`string`** and **`none`** for `TYPE` on string keys and missing keys as in `docs/type.md`.
