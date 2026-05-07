package commands

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

// PING returns PONG.
func PING() []byte {
	return resp.WriteSimpleString("PONG")
}

// ECHO returns the first argument as a bulk string.
func ECHO(args []string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'ECHO'")
	}
	return resp.WriteBulkString(args[0]), nil
}

// SET stores a string value, with optional PX expiry in milliseconds.
func SET(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'SET'")
	}

	key := args[0]
	value := args[1]

	mutex.Lock()
	cache[key] = value
	mutex.Unlock()

	if len(args) > 2 && args[2] == "PX" {
		duration, err := strconv.ParseInt(args[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid argument for 'SET'")
		}
		go deleteKeyAfterDuration(key, duration)
	}
	return resp.WriteSimpleString("OK"), nil
}

// GET returns the string value stored at key.
func GET(args []string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'GET'")
	}
	key := args[0]

	mutex.Lock()
	defer mutex.Unlock()

	raw, ok := cache[key]
	if !ok {
		return []byte("$-1" + resp.CRLF), nil
	}
	s, ok := raw.(string)
	if !ok {
		return nil, fmt.Errorf("GET: stored value for %q is not a string", key)
	}
	return resp.WriteBulkString(s), nil
}

// INCR increments the integer stored at key by one.
func INCR(args []string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'INCR'")
	}

	key := args[0]

	mutex.Lock()
	defer mutex.Unlock()

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

func deleteKeyAfterDuration(key string, duration int64) {
	time.Sleep(time.Duration(duration) * time.Millisecond)
	mutex.Lock()
	delete(cache, key)
	mutex.Unlock()
}
