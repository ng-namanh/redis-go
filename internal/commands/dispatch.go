package commands

import "fmt"

func HandleCommand(cmd string, args []string) ([]byte, error) {
	switch cmd {
	case "PING":
		return PING(), nil
	case "ECHO":
		return ECHO(args)
	case "SET":
		return SET(args)
	case "GET":
		return GET(args)
	case "INCR":
		return INCR(args)
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
	default:
		return nil, fmt.Errorf("unknown command '%s'", cmd)
	}
}
