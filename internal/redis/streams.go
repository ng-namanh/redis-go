package redis

import (
	"fmt"

	"github.com/ng-namanh/redis-go/internal/resp"
)

// Minimal Stream storage for XADD (docs/xadd.md): explicit IDs only, no MAXLEN/MINID options.
type Stream struct {
	entries []StreamEntry
}

type StreamEntry struct {
	id     string
	fields []string // flat k,v,k,v,...
}

var streams = make(map[string]*Stream)

func XADD(args []string) ([]byte, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("wrong number of arguments for 'XADD'")
	}
	if (len(args)-2)%2 != 0 {
		return nil, fmt.Errorf("wrong number of arguments for 'XADD'")
	}

	streamKey := args[0]
	id := args[1]
	fields := args[2:]

	listMu.Lock()
	defer listMu.Unlock()

	s := streams[streamKey]
	if s == nil {
		s = &Stream{}          // create new stream if it doesn't exist
		streams[streamKey] = s // add to streams map
	}

	// add new entry to stream
	s.entries = append(s.entries, StreamEntry{
		id:     id,
		fields: append([]string(nil), fields...),
	})
	return resp.WriteBulkString(id), nil
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
