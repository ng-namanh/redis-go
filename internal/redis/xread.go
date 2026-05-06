package redis

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

// XREAD reads entries from one or more streams, returning entries with IDs strictly greater
// than the caller-provided IDs. It supports COUNT, BLOCK, and the special IDs "$" and "+".
func XREAD(args []string) ([]byte, error) {
	count := -1
	blockMs := int64(-1)

	// Parse options until STREAMS.
	i := 0
	for i < len(args) {
		switch strings.ToUpper(args[i]) {
		case "COUNT":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("wrong number of arguments for 'XREAD'")
			}
			// convert the COUNT argument to an integer
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid COUNT argument for 'XREAD'")
			}
			count = n
			// we have processed the COUNT argument, so we move to the next argument
			i += 2
		case "BLOCK":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("wrong number of arguments for 'XREAD'")
			}

			// convert the BLOCK argument to an integer
			n, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid BLOCK argument for 'XREAD'")
			}
			blockMs = n
			// we have processed the BLOCK argument, so we move to the next argument
			i += 2
		case "STREAMS":
			// we have processed the STREAMS argument, so we move to the next argument
			i++
			goto afterOptions
		default:
			return nil, fmt.Errorf("wrong number of arguments for 'XREAD'")
		}
	}

afterOptions:
	if i >= len(args) {
		return nil, fmt.Errorf("wrong number of arguments for 'XREAD'")
	}

	// check if the number of arguments is valid
	remaining := args[i:]
	if len(remaining) < 2 || len(remaining)%2 != 0 {
		return nil, fmt.Errorf("wrong number of arguments for 'XREAD'")
	}

	// split the remaining arguments into keys and ids
	n := len(remaining) / 2
	keys := remaining[:n]
	ids := remaining[n:]

	type streamReq struct {
		key      string
		plus     bool
		lastSeen StreamId // used when plus == false
		valid    bool
	}

	// Snapshot "$" IDs at command invocation time (important for BLOCK semantics).
	reqs := make([]streamReq, 0, n)
	listMu.Lock()
	for si, key := range keys {
		token := ids[si]
		if token == "+" {
			reqs = append(reqs, streamReq{key: key, plus: true, valid: true})
			continue
		}

		if token == "$" {
			s := streams[key]
			// "$" means "from now": snapshot current last ID; if stream missing/empty, use 0-0.
			last := LastStreamEntryID(s)
			if last == "" {
				reqs = append(reqs, streamReq{key: key, lastSeen: StreamId{Ms: 0, Seq: 0}, valid: true})
				continue
			}
			id, ok := ParseStreamID(last)
			reqs = append(reqs, streamReq{key: key, lastSeen: id, valid: ok})
			continue
		}

		lastSeen, ok, err := parseXREADLastSeenID(token)
		if err != nil {
			listMu.Unlock()
			return nil, err
		}
		reqs = append(reqs, streamReq{key: key, lastSeen: lastSeen, valid: ok})
	}
	listMu.Unlock()

	for _, r := range reqs {
		if !r.valid {
			return nil, fmt.Errorf("Invalid stream ID")
		}
	}

	tryOnce := func() ([]resp.RESP, bool, error) {
		listMu.Lock()
		defer listMu.Unlock()

		// WRONGTYPE checks.
		for _, r := range reqs {
			k := r.key
			if _, ok := lists[k]; ok {
				return nil, false, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
			}
			if raw, ok := cache[k]; ok {
				if _, ok := raw.(string); ok {
					return nil, false, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
				}
			}
		}

		out := make([]resp.RESP, 0)
		served := false

		for _, r := range reqs {
			s := streams[r.key]
			if s == nil || len(s.entries) == 0 {
				continue
			}

			if r.plus {
				last := s.entries[len(s.entries)-1]
				entry := encodeStreamEntry(last.id, last.fields)
				out = append(out, resp.RESP{
					Type: resp.Array,
					Elems: []resp.RESP{
						{Type: resp.BulkString, Str: r.key},
						{Type: resp.Array, Elems: []resp.RESP{entry}},
					},
				})
				served = true
				continue
			}

			entries := make([]resp.RESP, 0)
			for _, e := range s.entries {
				eid, ok := ParseStreamID(e.id)
				if !ok {
					continue
				}
				if eid.GreaterThan(r.lastSeen) {
					entries = append(entries, encodeStreamEntry(e.id, e.fields))
					if count >= 0 && len(entries) >= count {
						break
					}
				}
			}
			if len(entries) == 0 {
				continue
			}
			out = append(out, resp.RESP{
				Type: resp.Array,
				Elems: []resp.RESP{
					{Type: resp.BulkString, Str: r.key},
					{Type: resp.Array, Elems: entries},
				},
			})
			served = true
		}

		return out, served, nil
	}

	// Non-blocking attempt first.
	out, served, err := tryOnce()
	if err != nil {
		return nil, err
	}
	if served {
		return resp.WriteArray(out), nil
	}
	if blockMs < 0 {
		return resp.WriteArray(nil), nil
	}

	// Blocking mode: poll until data available or timeout.
	deadline := time.Time{}
	if blockMs > 0 {
		deadline = time.Now().Add(time.Duration(blockMs) * time.Millisecond)
	}

	for {
		out, served, err := tryOnce()
		if err != nil {
			return nil, err
		}
		if served {
			return resp.WriteArray(out), nil
		}
		if !deadline.IsZero() && time.Now().After(deadline) {
			// RESP2 Nil reply (null bulk string) on BLOCK timeout.
			return []byte("$-1" + resp.CRLF), nil
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func encodeStreamEntry(id string, fields []string) resp.RESP {
	fieldElems := make([]resp.RESP, 0, len(fields))
	for _, f := range fields {
		fieldElems = append(fieldElems, resp.RESP{Type: resp.BulkString, Str: f})
	}
	return resp.RESP{
		Type: resp.Array,
		Elems: []resp.RESP{
			{Type: resp.BulkString, Str: id},
			{Type: resp.Array, Elems: fieldElems},
		},
	}
}

// parseXREADLastSeenID parses the "last seen" ID token for XREAD for a single stream.
// It supports:
//   - full "<ms>-<seq>"
//   - milliseconds-only "<ms>" (sequence is interpreted as 0 for XREAD)
func parseXREADLastSeenID(token string) (StreamId, bool, error) {
	if strings.Contains(token, "-") {
		id, ok := ParseStreamID(token)
		return id, ok, nil
	}
	if token == "" {
		return StreamId{}, false, nil
	}
	ms, err := strconv.ParseUint(token, 10, 64)
	if err != nil {
		return StreamId{}, false, nil
	}
	return StreamId{Ms: ms, Seq: 0}, true, nil
}
