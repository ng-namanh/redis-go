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
		return resp.WriteSimpleString("PONG"), nil
	case "ECHO":
		if len(args) < 1 {
			return nil, fmt.Errorf("wrong number of arguments for 'ECHO'")
		}
		return resp.WriteBulkString(args[0]), nil
	case "SET":

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
	case "GET":
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
	case "RPUSH":
		if len(args) < 2 {
			return nil, fmt.Errorf("wrong number of arguments for 'RPUSH'")
		}

		listName := args[0]

		if _, ok := lists[listName]; !ok {
			lists[listName] = make(list, 0)
		}

		lists[listName] = append(lists[listName], args[1:]...)

		return resp.WriteInteger(int64(len(lists[listName]))), nil
	case "LRANGE":
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

	default:
		return nil, fmt.Errorf("unknown command '%s'", cmd)
	}
}

func deleteKeyAfterDuration(key string, duration int64) {
	time.Sleep(time.Duration(duration) * time.Millisecond)
	delete(cache, key)
}
