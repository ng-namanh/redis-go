# XADD — pseudo implementation

This document is a **language-agnostic** sketch of how `XADD` is implemented in this project (explicit stream IDs, field pairs, ID validation). It matches the rules in `xadd.md`.

---

## Data model (conceptual)

```text
STREAMS: map<streamKey, Stream>
Stream: { entries: [ StreamEntry, ... ] }
StreamEntry: { id: string, fields: [ string, string, ... ] }  // fields = flat k1,v1,k2,v2,...
```

---

## `parseStreamID(id) -> (ms, seq) | invalid`

```text
function parseStreamID(id):
  i = index of first '-' in id
  if i is missing, or i == 0, or i is last index:
    return invalid

  msStr = id[0 .. i-1]
  seqStr = id[i+1 .. end]
  if seqStr contains another '-':
    return invalid   // only one separator: <ms>-<seq>

  if msStr or seqStr is not a non-negative decimal integer:
    return invalid

  return (ms, seq) as unsigned integers
```

---

## `newId > lastId` (strictly greater)

```text
function idGreaterThan(new, last):
  return (new.ms > last.ms)
     OR (new.ms == last.ms AND new.seq > last.seq)
```

---

## `0-0` rule

```text
function isZeroId(id):
  return id.ms == 0 AND id.seq == 0
```

`0-0` is **always** invalid; respond with a simple error whose text includes that the ID must be greater than `0-0`.

---

## `XADD(args) -> reply bytes or error`

**Arguments (this stage):** `streamKey`, `idString`, then an even number of strings `field, value, ...`

```text
function XADD(args):
  if len(args) < 4:
    return arity error
  if (len(args) - 2) is odd:
    return arity error   // need full field-value pairs after key and id

  streamKey = args[0]
  idString  = args[1]
  fields    = args[2 .. end]

  newId = parseStreamID(idString)
  if newId is invalid:
    return command error (invalid stream ID)   // or your project’s error policy

  if isZeroId(newId):
    return simple error: "ID must be greater than 0-0"

  lock(streamKey)   // same mutex as other in-memory keyspace in this server

  stream = STREAMS[streamKey]
  if stream exists AND stream.entries is not empty:
    lastIdString = last entry’s id in stream
    lastId = parseStreamID(lastIdString)
    if lastId is invalid:
      return command error
    if NOT idGreaterThan(newId, lastId):
      return simple error: "ID ... equal or smaller than the target stream top item"

  if stream is nil:
    stream = new empty Stream
    STREAMS[streamKey] = stream

  append to stream.entries: { id: idString, fields: copy(fields) }

  unlock()

  return RESP bulk string idString
```

**Success reply:** the new entry’s ID as a **bulk string** (e.g. Redis `XADD` returns the ID the client can display in quotes).

---

## Auto ID when `idString == "*"`

```text
function nextAutoStreamID(stream):
  now = current Unix time in milliseconds (unsigned)

  if stream is nil or stream.entries is empty:
    return format(now, 0)

  last = parseStreamID(last entry id)
  if last is invalid:
    return format(now, 0)

  if now > last.ms:
    return format(now, 0)
  if now == last.ms:
    return format(now, last.seq + 1)
  // clock behind last entry
  return format(last.ms, last.seq + 1)
```

Inside `XADD`, if the client passes `*`, set `finalID = nextAutoStreamID(s)` under the same lock **before** appending (so two `*` in a row see the updated last entry).

---

## Partial auto: `idString == "<ms>-*"`

```text
function nextPartialSeqStreamID(stream, ms):
  if stream is nil or empty:
    if ms == 0:
      return format(0, 1)   // must be > 0-0
    return format(ms, 0)

  last = parseStreamID(last entry id)
  if ms < last.ms:
    return error "equal or smaller than top"
  if ms > last.ms:
    return format(ms, 0)
  return format(ms, last.seq + 1)
```

---

## Threading / locking (this codebase)

- All access to `STREAMS` and `Stream.entries` for `XADD` / `TYPE` on stream keys is done under the same **mutex** used for list and blocking list state, so one mutates the store at a time and `TYPE` sees a consistent view.

---

## Not in this pseudo (later stages)

- `MAXLEN`, `NOMKSTREAM`, `MINID`, etc.
