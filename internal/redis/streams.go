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

// TYPE implements Redis TYPE per docs/type.md: simple string reply with the type name, or "none".
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
