package redis

import (
	"fmt"
	"strconv"

	"github.com/ng-namanh/redis-go/internal/resp"
)

var cache = map[string]any{}

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
	case "BLPOP":
		return BLPOP(args)
	case "TYPE":
		return TYPE(args)
	case "XADD":
		return XADD(args)
	case "XRANGE":
		return XRANGE(args)
	case "XREAD":
		return XREAD(args)
	case "INCR":
		return INCR(args)
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
