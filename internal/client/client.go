package client

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/ng-namanh/redis-go/internal/redis"
	"github.com/ng-namanh/redis-go/internal/resp"
)

type QueuedCommand struct {
	Name string
	Args []string
}

type Client struct {
	InMulti        bool
	QueuedCommands []QueuedCommand
}

func NewClient() *Client {
	return &Client{}
}

func (c *Client) DispatchCommand(v resp.RESP) ([]byte, error) {
	cmd, args, err := resp.ParseCommand(v)
	if err != nil {
		return nil, err
	}

	if c.InMulti && cmd != "EXEC" && cmd != "DISCARD" && cmd != "MULTI" {
		// add command to command queue
		c.QueuedCommands = append(c.QueuedCommands, QueuedCommand{Name: cmd, Args: args})
		return resp.WriteSimpleString("QUEUED"), nil
	}

	return c.HandleCommand(cmd, args)
}

func (c *Client) HandleCommand(cmd string, args []string) ([]byte, error) {
	switch cmd {
	case "MULTI":
		return c.MULTI(args)
	case "EXEC":
		return c.EXEC(args)
	case "DISCARD":
		return c.DISCARD(args)
	case "PING":
		return redis.PING(), nil
	case "ECHO":
		return redis.ECHO(args)
	case "SET":
		return redis.SET(args)
	case "GET":
		return redis.GET(args)
	case "RPUSH":
		return redis.RPUSH(args)
	case "LPUSH":
		return redis.LPUSH(args)
	case "LRANGE":
		return redis.LRANGE(args)
	case "LLEN":
		return redis.LLEN(args)
	case "LPOP":
		return redis.LPOP(args)
	case "BLPOP":
		return redis.BLPOP(args)
	case "TYPE":
		return redis.TYPE(args)
	case "XADD":
		return redis.XADD(args)
	case "XRANGE":
		return redis.XRANGE(args)
	case "XREAD":
		return redis.XREAD(args)
	case "INCR":
		return redis.INCR(args)
	default:
		return nil, fmt.Errorf("unknown command '%s'", cmd)
	}
}

func (c *Client) MULTI(args []string) ([]byte, error) {
	if c.InMulti {
		return nil, fmt.Errorf("MULTI calls can not be nested")
	}
	c.InMulti = true
	c.QueuedCommands = nil
	return resp.WriteSimpleString("OK"), nil
}

func (c *Client) DISCARD(args []string) ([]byte, error) {
	if !c.InMulti {
		return nil, fmt.Errorf("DISCARD without MULTI")
	}
	c.InMulti = false
	c.QueuedCommands = nil
	return resp.WriteSimpleString("OK"), nil
}

func (c *Client) EXEC(args []string) ([]byte, error) {
	if !c.InMulti {
		return nil, fmt.Errorf("EXEC without MULTI")
	}

	cmds := c.QueuedCommands
	c.InMulti = false
	c.QueuedCommands = nil

	if len(cmds) == 0 {
		return resp.WriteArray(nil), nil
	}

	results := make([]resp.RESP, 0, len(cmds))
	for _, q := range cmds {
		out, err := c.HandleCommand(q.Name, q.Args)
		if err != nil {
			results = append(results, resp.RESP{Type: resp.Error, Err: "ERR " + err.Error()})
			continue
		}

		// Parse the output back to RESP
		r := bufio.NewReader(bytes.NewReader(out))
		val, err := resp.ReadValue(r)
		if err != nil {
			results = append(results, resp.RESP{Type: resp.Error, Err: "ERR " + err.Error()})
		} else {
			results = append(results, val)
		}
	}

	return resp.WriteArray(results), nil
}
