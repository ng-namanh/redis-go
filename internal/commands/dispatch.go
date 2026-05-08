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
	case "INFO":
		return INFO(args)
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

// HandleCommandUnlocked executes a command without acquiring the global lock.
// This is used for atomic transactions where the caller already holds the lock.
func HandleCommandUnlocked(cmd string, args []string) ([]byte, error) {
	switch cmd {
	case "PING":
		return PING(), nil
	case "ECHO":
		return ECHO(args)
	case "SET":
		return setUnlocked(args)
	case "GET":
		return getUnlocked(args)
	case "INCR":
		return incrUnlocked(args)
	case "INFO":
		return INFO(args)
	case "RPUSH":
		return rpushUnlocked(args)
	case "LPUSH":
		return lpushUnlocked(args)
	case "LRANGE":
		return lrangeUnlocked(args)
	case "LLEN":
		return llenUnlocked(args)
	case "LPOP":
		return lpopUnlocked(args)
	case "TYPE":
		return typeUnlocked(args)
	case "XADD":
		return xaddUnlocked(args)
	case "XRANGE":
		return xrangeUnlocked(args)
	default:
		// Commands that are still blocking or not yet refactored will fallback to HandleCommand.
		// Note: This may cause deadlocks if those commands attempt to Lock().
		return HandleCommand(cmd, args)
	}
}
