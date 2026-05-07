// Package client manages per-connection state (transaction queue, etc.)
// and routes incoming RESP commands to their implementations in the commands package.
package client

import (
	"fmt"

	"github.com/ng-namanh/redis-go/internal/commands"
	"github.com/ng-namanh/redis-go/internal/resp"
)

// Client holds the per-connection state for a single Redis client session.
type Client struct {
	inMulti        bool
	queuedCommands []QueuedCommand
}

// NewClient returns a new, ready-to-use Client.
func NewClient() *Client {
	return &Client{}
}

// DispatchCommand parses v and either queues it (when inside a MULTI block) or
// executes it immediately.
func (c *Client) DispatchCommand(v resp.RESP) ([]byte, error) {
	cmd, args, err := resp.ParseCommand(v)
	if err != nil {
		return nil, err
	}

	if c.inMulti && cmd != "EXEC" && cmd != "DISCARD" && cmd != "MULTI" {
		// add command to command queue
		c.queuedCommands = append(c.queuedCommands, QueuedCommand{Name: cmd, Args: args})
		return resp.WriteSimpleString("QUEUED"), nil
	}

	return c.HandleCommand(cmd, args)
}

// HandleCommand executes a single command by name, bypassing the queuing logic.
// Used both by DispatchCommand and internally by EXEC.
func (c *Client) HandleCommand(cmd string, args []string) ([]byte, error) {
	switch cmd {
	// Transaction commands (need client state).
	case "MULTI":
		return c.multi(args)
	case "EXEC":
		return c.exec(args)
	case "DISCARD":
		return c.discard(args)

	// General commands (stateless — delegate to commands package).
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
