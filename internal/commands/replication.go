package commands

import (
	"fmt"
	"strings"

	"github.com/ng-namanh/redis-go/internal/resp"
)

func INFO(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()

	section := ""
	if len(args) > 0 {
		section = strings.ToLower(args[0])
	}

	var sb strings.Builder
	if section == "" || section == "replication" {
		sb.WriteString(fmt.Sprintf("role:%s\r\n", Role))
	}

	return resp.WriteBulkString(sb.String()), nil
}
