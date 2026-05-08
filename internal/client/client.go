package client

import (
	"fmt"

	"github.com/ng-namanh/redis-go/internal/commands"
	"github.com/ng-namanh/redis-go/internal/resp"
)

type Client struct {
	inMulti        bool
	queuedCommands []QueuedCommand
	watchedKeys    map[string]uint64 // key -> version when WATCHed
}

func NewClient() *Client {
	return &Client{
		watchedKeys: make(map[string]uint64),
	}
}

func (c *Client) DispatchCommand(v resp.RESP) ([]byte, error) {
	cmd, args, err := resp.ParseCommand(v)
	if err != nil {
		return nil, err
	}

	if c.inMulti && cmd != "EXEC" && cmd != "DISCARD" && cmd != "MULTI" {
		c.queuedCommands = append(c.queuedCommands, QueuedCommand{Name: cmd, Args: args})
		return resp.WriteSimpleString("QUEUED"), nil
	}

	return c.HandleCommand(cmd, args)
}

func (c *Client) HandleCommand(cmd string, args []string) ([]byte, error) {
	switch cmd {

	// Transactions
	case "MULTI":
		return c.multi(args)
	case "EXEC":
		return c.exec(args)
	case "DISCARD":
		return c.discard(args)
	case "WATCH":
		return c.watch(args)
	case "UNWATCH":
		return c.unwatch(args)

	case "PING":
		return commands.PING(), nil
	case "ECHO":
		return commands.ECHO(args)
	case "SET":
		return commands.SET(args)
	case "GET":
		return commands.GET(args)
	case "INCR":
		return commands.INCR(args)

	// List commands.
	case "RPUSH":
		return commands.RPUSH(args)
	case "LPUSH":
		return commands.LPUSH(args)
	case "LRANGE":
		return commands.LRANGE(args)
	case "LLEN":
		return commands.LLEN(args)
	case "LPOP":
		return commands.LPOP(args)
	case "BLPOP":
		return commands.BLPOP(args)

	// Stream commands.
	case "TYPE":
		return commands.TYPE(args)
	case "XADD":
		return commands.XADD(args)
	case "XRANGE":
		return commands.XRANGE(args)
	case "XREAD":
		return commands.XREAD(args)

	default:
		return nil, fmt.Errorf("unknown command '%s'", cmd)
	}
}
