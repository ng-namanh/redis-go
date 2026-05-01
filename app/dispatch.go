package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

type list []string

var cache = map[string]any{}
var lists map[string]list = make(map[string]list)

// lrangeSlice returns Redis LRANGE-inclusive elements for start/stop (negative indices OK).
func lrangeSlice(l list, start, stop int) list {
	ln := len(l)
	if ln == 0 {
		return nil
	}

	startIdx := start

	if startIdx >= ln {
		return nil
	}

	if startIdx < 0 {
		startIdx += ln

		// if the start index is negative, and the absolute value of the start index is greater than the length of the list, set the start index to 0
		// Example: start = -12, ln = 10, startIdx = -12 + 10 = -2 -> startIdx = 0
		if startIdx < 0 {
			startIdx = 0
		}
	}

	stopIdx := stop
	if stopIdx < 0 {
		stopIdx += ln
		if stopIdx < 0 {
			stopIdx = 0
		}
	}

	if stopIdx >= ln {
		stopIdx = ln - 1
	}

	if startIdx > stopIdx {
		return nil
	}

	return l[startIdx : stopIdx+1]
}

func DispatchCommand(v resp.RESP) ([]byte, error) {
	cmd, args, err := resp.ParseCommand(v)
	if err != nil {
		return nil, err
	}

	switch cmd {
	case "PING":
		return PING(), nil
	case "ECHO":
		return ECHO(args)
	case "SET":
		return SET(args)
	case "GET":
		return GET(args)
	case "RPUSH":
		return RPUSH(args)
	case "LPUSH":
		return LPUSH(args)
	case "LRANGE":
		return LRANGE(args)
	case "LLEN":
		return LLEN(args)
	case "LPOP":
		return LPOP(args)
	default:
		return nil, fmt.Errorf("unknown command '%s'", cmd)
	}
}

func PING() []byte {
	return resp.WriteSimpleString("PONG")
}

func ECHO(args []string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'ECHO'")
	}
	return resp.WriteBulkString(args[0]), nil
}

func SET(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'SET'")
	}

	key := args[0]
	value := args[1]

	cache[key] = value
	if len(args) > 2 && args[2] == "PX" {
		duration, err := strconv.ParseInt(args[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid argument for 'SET'")
		}
		go deleteKeyAfterDuration(key, duration)
	}
	return resp.WriteSimpleString("OK"), nil
}

func GET(args []string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'GET'")
	}
	key := args[0]
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

func RPUSH(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'RPUSH'")
	}

	listName := args[0]

	if _, ok := lists[listName]; !ok {
		lists[listName] = make(list, 0)
	}

	lists[listName] = append(lists[listName], args[1:]...)

	return resp.WriteInteger(int64(len(lists[listName]))), nil
}

func LRANGE(args []string) ([]byte, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'LRANGE'")
	}

	listName := args[0]
	start, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, fmt.Errorf("invalid argument for 'LRANGE'")
	}
	end, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, fmt.Errorf("invalid argument for 'LRANGE'")
	}

	sub := lrangeSlice(lists[listName], start, end)
	elements := make([]resp.RESP, 0, len(sub))
	for _, s := range sub {
		elements = append(elements, resp.RESP{Type: resp.BulkString, Str: s})
	}
	return resp.WriteArray(elements), nil
}

func LPUSH(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'LPUSH'")
	}

	listName := args[0]
	vals := args[1:]
	old := lists[listName]

	// Redis: each value is pushed onto the head in argument order, so last arg is the new head.
	// Do not use args[1:] as the first operand to append — it can alias the request buffer.
	n := len(vals) + len(old)
	prefix := make(list, n)
	at := 0
	for i := len(vals) - 1; i >= 0; i-- {
		prefix[at] = vals[i]
		at++
	}
	copy(prefix[at:], old)
	lists[listName] = prefix

	return resp.WriteInteger(int64(len(lists[listName]))), nil
}

func LLEN(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'LLEN'")
	}

	listName := args[0]
	if _, holdsKey := cache[listName]; holdsKey {
		return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return resp.WriteInteger(int64(len(lists[listName]))), nil
}

func LPOP(args []string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'LPOP'")
	}
	if len(args) > 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'LPOP'")
	}

	key := args[0]
	if _, inCache := cache[key]; inCache {
		return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	lst := lists[key]
	if len(lst) == 0 {
		return []byte("$-1" + resp.CRLF), nil
	}

	// remove first element from the selected list
	if len(args) == 1 {
		v := lst[0]
		lists[key] = lst[1:]
		if len(lists[key]) == 0 {
			delete(lists, key)
		}
		return resp.WriteBulkString(v), nil
	}

	// get the count of elements to remove
	count, err := strconv.Atoi(args[1])
	if err != nil || count <= 0 {
		return nil, fmt.Errorf("invalid argument for 'LPOP'")
	}

	// remove first `count` elements from the selected list
	n := min(count, len(lst)) // if count is greater than the length of the list, remove all elements
	popped := lst[:n]

	// append the remaining elements to the selected list
	// Do this because the selected list is a slice of the original list, and we need to update the original list.
	lists[key] = append(list(nil), lst[n:]...)

	// if the selected list is empty, delete the key
	// Why do we need to delete the selected list from the lists map?
	// If we don't delete the selected list from the lists map, the original list will still be in the maps, and we will have a memory leak.
	if len(lists[key]) == 0 {
		delete(lists, key)
	}

	out := make([]resp.RESP, 0, len(popped))

	// Build the response array to convert the popped elements to RESP Bulk Strings object
	for _, s := range popped {
		out = append(out, resp.RESP{Type: resp.BulkString, Str: s})
	}
	return resp.WriteArray(out), nil
}

func deleteKeyAfterDuration(key string, duration int64) {
	time.Sleep(time.Duration(duration) * time.Millisecond)
	delete(cache, key)
}
