package commands

import (
	"strings"

	"github.com/ng-namanh/redis-go/internal/resp"
)

func CONFIG(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, nil
	}

	subcommand := strings.ToUpper(args[0])
	if subcommand != "GET" {
		return nil, nil
	}

	key := strings.ToLower(args[1])
	var val string

	switch key {
	case "dir":
		val = Dir
	case "dbfilename":
		val = Dbfilename
	case "port":
		val = ServerPort
	default:
		return resp.WriteArray([]resp.RESP{}), nil
	}

	return resp.WriteArray([]resp.RESP{
		{Type: resp.BulkString, Str: key},
		{Type: resp.BulkString, Str: val},
	}), nil
}
