package commands

import (
	"errors"
	"fmt"
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

// XADD appends an entry to a stream key and returns the final entry ID.
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

	mutex.Lock()
	defer mutex.Unlock()

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

// XRANGE returns stream entries in the inclusive [start, end] ID range.
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
	if !StreamIdGte(end, start) {
		return resp.WriteArray(nil), nil
	}

	mutex.Lock()
	defer mutex.Unlock()

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

// TYPE returns the type of the value stored at key.
func TYPE(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'TYPE'")
	}
	key := args[0]

	mutex.Lock()
	defer mutex.Unlock()

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
