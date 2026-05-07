// Package redis is a compatibility shim. All command implementations live in
// internal/commands; this package re-exports the symbols that tests and other
// callers depend on.
package redis

import (
	"github.com/ng-namanh/redis-go/internal/commands"
	"github.com/ng-namanh/redis-go/internal/resp"
)

type StreamId = commands.StreamId

var (
	ParseStreamID = commands.ParseStreamID
	XADD          = commands.XADD
	TYPE          = commands.TYPE
)

func DispatchCommand(v resp.RESP) ([]byte, error) {
	cmd, args, err := resp.ParseCommand(v)
	if err != nil {
		return nil, err
	}
	return commands.HandleCommand(cmd, args)
}

func ResetForTesting() {
	commands.ResetForTesting()
}
