package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

type store map[string]any

var db store = make(store)

func DispatchCommand(v resp.RESP) ([]byte, error) {
	cmd, args, err := resp.ParseCommand(v)
	if err != nil {
		return nil, err
	}

	switch cmd {
	case "PING":
		return resp.AppendSimpleString("PONG"), nil
	case "ECHO":
		if len(args) < 1 {
			return nil, fmt.Errorf("wrong number of arguments for 'ECHO'")
		}
		return resp.AppendBulkString(args[0]), nil
	case "SET":

		if len(args) < 2 {
			return nil, fmt.Errorf("wrong number of arguments for 'SET'")
		}

		key := args[0]
		value := args[1]

		db[key] = value
		if len(args) > 2 && args[2] == "PX" {
			duration, err := strconv.ParseInt(args[3], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid argument for 'SET'")
			}
			go deleteKeyAfterDuration(key, duration)
		}
		return resp.AppendSimpleString("OK"), nil
	case "GET":
		if len(args) < 1 {
			return nil, fmt.Errorf("wrong number of arguments for 'GET'")
		}
		key := args[0]
		raw, ok := db[key]
		if !ok {
			return []byte("$-1" + resp.CRLF), nil
		}
		s, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("GET: stored value for %q is not a string", key)
		}
		return resp.AppendBulkString(s), nil
	default:
		return nil, fmt.Errorf("unknown command '%s'", cmd)
	}
}

func deleteKeyAfterDuration(key string, duration int64) {
	time.Sleep(time.Duration(duration) * time.Millisecond)
	delete(db, key)
}
