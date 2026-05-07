package redis

import (
	"fmt"

	"github.com/ng-namanh/redis-go/internal/resp"
)

func INCR(args []string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'INCR'")
	}

	key := args[0]

	listMu.Lock()
	defer listMu.Unlock()

	// get the value from the cache
	val, ok := cache[key]
	if !ok {
		cache[key] = 1
		return resp.WriteInteger(1), nil
	}

	s, ok := val.(int)
	if !ok {
		return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	s++
	cache[key] = s
	return resp.WriteInteger(int64(s)), nil
}
