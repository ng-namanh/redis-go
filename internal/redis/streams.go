package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

var streams = make(map[string]*Stream)

type Stream struct {
	entries []StreamEntry
}

type StreamEntry struct {
	id     string
	fields []string // flat k,v,k,v,...
}

// XADD appends an entry to a stream key and returns the final entry ID.
// It supports Redis-style auto IDs ("*") and partial sequence auto IDs ("<ms>-*"),
// and enforces monotonically increasing IDs per stream.
func XADD(args []string) ([]byte, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("wrong number of arguments for 'XADD'")
	}
	if (len(args)-2)%2 != 0 {
		return nil, fmt.Errorf("wrong number of arguments for 'XADD'")
	}

	streamKey := args[0]
	idStr := args[1]
	fields := args[2:]

	listMu.Lock()
	defer listMu.Unlock()

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
	return resp.WriteBulkString(finalID), nil
}

// XRANGE returns stream entries in the inclusive [start, end] ID range for a stream key.
// If the key doesn't exist, the range is invalid, or end < start, it returns an empty array.
func XRANGE(args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'XRANGE'")
	}
	key, startStr, endStr := args[0], args[1], args[2]
	start, ok1 := ParseXRANGEBound(startStr, true)
	end, ok2 := ParseXRANGEBound(endStr, false)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("Invalid stream ID")
	}

	// If end is < start, return empty array
	if !StreamIdGte(end, start) {
		return resp.WriteArray(nil), nil
	}

	listMu.Lock()
	defer listMu.Unlock()

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
		return resp.WriteArray(nil), nil
	}

	out := make([]resp.RESP, 0)

	// Iterate over the stream entries and add them to the output array if they are in the range
	for _, e := range s.entries {
		eid, ok := ParseStreamID(e.id)

		// skip this entry if the ID is invalid
		if !ok {
			continue
		}

		// skip this entry if the ID is not in the range
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

// TYPE implements Redis TYPE (per docs/type.md): returns the key's type name, or "none".
// Supported in this server: stream (XADD), list, string (SET/GET cache).
func TYPE(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'TYPE'")
	}

	key := args[0]

	listMu.Lock()
	defer listMu.Unlock()

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
