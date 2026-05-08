package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

const (
	ErrXADDIDNotGreaterThanTop = "ERR The ID specified in XADD is equal or smaller than the target stream top item"
	ErrXADDIDMustBeGreater0    = "ERR The ID specified in XADD must be greater than 0-0"
)

// ErrNotGreaterThanTop means the proposed ID is not strictly greater than the stream's last entry.
var ErrNotGreaterThanTop = errors.New("stream id not greater than top")

type Stream struct {
	entries []StreamEntry
}

type StreamEntry struct {
	id     string
	fields []string // flat k,v,k,v,...
}

func XADD(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return xaddUnlocked(args)
}

func xaddUnlocked(args []string) ([]byte, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("wrong number of arguments for 'XADD'")
	}
	if (len(args)-2)%2 != 0 {
		return nil, fmt.Errorf("wrong number of arguments for 'XADD'")
	}

	streamKey := args[0]
	idStr := args[1]
	fields := args[2:]

	s := streams[streamKey]
	lastID := LastStreamEntryID(s)

	var finalID string

	if idStr == "*" {
		finalID = NextAutoFull(uint64(time.Now().UnixMilli()), lastID)
	} else if pms, ok := ParsePartialSeqAutoID(idStr); ok {
		id, err := NextPartialSeqStreamID(pms, lastID)
		if err != nil {
			if errors.Is(err, ErrNotGreaterThanTop) {
				return resp.WriteError(ErrXADDIDNotGreaterThanTop), nil
			}
			return nil, err
		}
		finalID = id
	} else {
		newID, ok := ParseStreamID(idStr)
		if !ok {
			return nil, fmt.Errorf("Invalid stream ID")
		}
		if newID.Ms == 0 && newID.Seq == 0 {
			return resp.WriteError(ErrXADDIDMustBeGreater0), nil
		}
		if lastID != "" {
			lastStreamID, ok := ParseStreamID(lastID)
			if !ok {
				return nil, fmt.Errorf("Invalid stream ID")
			}
			if !newID.GreaterThan(lastStreamID) {
				return resp.WriteError(ErrXADDIDNotGreaterThanTop), nil
			}
		}
		finalID = idStr
	}

	if s == nil {
		s = &Stream{}
		streams[streamKey] = s
	}
	s.entries = append(s.entries, StreamEntry{
		id:     finalID,
		fields: append([]string(nil), fields...),
	})
	Touch(streamKey)
	return resp.WriteBulkString(finalID), nil
}

func XRANGE(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return xrangeUnlocked(args)
}

func xrangeUnlocked(args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'XRANGE'")
	}
	key, startStr, endStr := args[0], args[1], args[2]
	start, ok1 := ParseXRANGEBound(startStr, true)
	end, ok2 := ParseXRANGEBound(endStr, false)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("Invalid stream ID")
	}
	if !StreamIdGte(end, start) {
		return resp.WriteArray([]resp.RESP{}), nil
	}

	if _, ok := lists[key]; ok {
		return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	if raw, ok := cache[key]; ok {
		if _, ok := raw.(string); ok {
			return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	}

	s := streams[key]
	if s == nil {
		return resp.WriteArray([]resp.RESP{}), nil
	}

	out := make([]resp.RESP, 0)
	for _, e := range s.entries {
		eid, ok := ParseStreamID(e.id)
		if !ok {
			continue
		}
		if !StreamIdGte(eid, start) || !StreamIdLte(eid, end) {
			continue
		}
		fieldElems := make([]resp.RESP, 0, len(e.fields))
		for _, f := range e.fields {
			fieldElems = append(fieldElems, resp.RESP{Type: resp.BulkString, Str: f})
		}
		out = append(out, resp.RESP{
			Type: resp.Array,
			Elems: []resp.RESP{
				{Type: resp.BulkString, Str: e.id},
				{Type: resp.Array, Elems: fieldElems},
			},
		})
	}
	return resp.WriteArray(out), nil
}

func TYPE(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return typeUnlocked(args)
}

func typeUnlocked(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'TYPE'")
	}
	key := args[0]

	if _, ok := streams[key]; ok {
		return resp.WriteSimpleString("stream"), nil
	}
	if _, ok := lists[key]; ok {
		return resp.WriteSimpleString("list"), nil
	}
	if raw, ok := cache[key]; ok {
		if _, ok := raw.(string); ok {
			return resp.WriteSimpleString("string"), nil
		}
	}
	return resp.WriteSimpleString("none"), nil
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

func XREAD(args []string) ([]byte, error) {
	count := -1
	blockMs := int64(-1)

	// Parse options until STREAMS keyword.
	i := 0
	for i < len(args) {
		switch strings.ToUpper(args[i]) {
		case "COUNT":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("wrong number of arguments for 'XREAD'")
			}
			n, err := strconv.Atoi(args[i+1])
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid COUNT argument for 'XREAD'")
			}
			count = n
			i += 2
		case "BLOCK":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("wrong number of arguments for 'XREAD'")
			}
			n, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("invalid BLOCK argument for 'XREAD'")
			}
			blockMs = n
			i += 2
		case "STREAMS":
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

	remaining := args[i:]
	if len(remaining) < 2 || len(remaining)%2 != 0 {
		return nil, fmt.Errorf("wrong number of arguments for 'XREAD'")
	}

	n := len(remaining) / 2
	keys := remaining[:n]
	ids := remaining[n:]

	type streamReq struct {
		key      string
		plus     bool
		lastSeen StreamId
		valid    bool
	}

	// Snapshot "$" IDs at command invocation time (important for BLOCK semantics).
	reqs := make([]streamReq, 0, n)
	mutex.Lock()

	for si, key := range keys {
		token := ids[si]
		if token == "+" {
			reqs = append(reqs, streamReq{key: key, plus: true, valid: true})
			continue
		}
		if token == "$" {
			s := streams[key]
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
			mutex.Unlock()
			return nil, err
		}
		reqs = append(reqs, streamReq{key: key, lastSeen: lastSeen, valid: ok})
	}
	mutex.Unlock()

	for _, r := range reqs {
		if !r.valid {
			return nil, fmt.Errorf("Invalid stream ID")
		}
	}

	tryOnce := func() ([]resp.RESP, bool, error) {
		mutex.Lock()
		defer mutex.Unlock()

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

	out, served, err := tryOnce()
	if err != nil {
		return nil, err
	}
	if served {
		return resp.WriteArray(out), nil
	}
	if blockMs < 0 {
		return resp.WriteArray([]resp.RESP{}), nil
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
			return []byte("$-1" + resp.CRLF), nil
		}
		time.Sleep(10 * time.Millisecond)
	}
}
