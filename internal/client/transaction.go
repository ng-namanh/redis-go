package client

import (
	"bufio"
	"bytes"
	"fmt"

	"github.com/ng-namanh/redis-go/internal/commands"
	"github.com/ng-namanh/redis-go/internal/resp"
)

// QueuedCommand is a command that has been queued inside a MULTI block.
type QueuedCommand struct {
	Name string
	Args []string
}

func (c *Client) multi(_ []string) ([]byte, error) {
	if c.inMulti {
		return nil, fmt.Errorf("MULTI calls can not be nested")
	}
	c.inMulti = true
	c.queuedCommands = nil
	return resp.WriteSimpleString("OK"), nil
}

func (c *Client) discard(_ []string) ([]byte, error) {
	if !c.inMulti {
		return nil, fmt.Errorf("DISCARD without MULTI")
	}
	c.inMulti = false
	c.queuedCommands = nil
	return resp.WriteSimpleString("OK"), nil
}

func (c *Client) exec(_ []string) ([]byte, error) {
	if !c.inMulti {
		return nil, fmt.Errorf("EXEC without MULTI")
	}

	cmds := c.queuedCommands
	c.inMulti = false
	c.queuedCommands = nil

	if len(cmds) == 0 {
		return resp.WriteArray(nil), nil
	}

	// Acquire the global lock for the entire duration of EXEC to ensure atomicity.
	commands.Lock()
	defer commands.Unlock()

	results := make([]resp.RESP, 0, len(cmds))
	for _, q := range cmds {
		// Use the "Unlocked" version of the dispatcher to avoid re-locking the same mutex.
		out, err := commands.HandleCommandUnlocked(q.Name, q.Args)
		if err != nil {
			results = append(results, resp.RESP{Type: resp.Error, Err: "ERR " + err.Error()})
			continue
		}

		// Re-parse the raw RESP bytes back into a RESP value so we can embed it
		// in the outer array reply.
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
