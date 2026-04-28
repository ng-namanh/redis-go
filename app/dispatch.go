package main

import (
	"fmt"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func DispatchCommand(v resp.Value) ([]byte, error) {
	cmd, args, err := resp.ParseCommand(v)
	if err != nil {
		return nil, err
	}

	switch cmd {
	case "PING":
		return resp.AppendSimpleString(nil, "PONG"), nil
	case "ECHO":
		if len(args) < 1 {
			return nil, fmt.Errorf("wrong number of arguments for 'echo'")
		}
		return resp.AppendBulkString(nil, args[0]), nil
	default:
		return nil, fmt.Errorf("unknown command '%s'", cmd)
	}
}
