package server

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/ng-namanh/redis-go/internal/client"
	"github.com/ng-namanh/redis-go/internal/commands"
	"github.com/ng-namanh/redis-go/internal/resp"
)

func Handle(conn net.Conn) {
	defer conn.Close()
	defer commands.RemoveSubscriber(conn)
	r := bufio.NewReader(conn)
	c := client.NewClient(conn)
	for {
		v, err := resp.ReadValue(r)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Println("Error reading RESP: ", err.Error())
			return
		}

		out, err := c.DispatchCommand(v)
		if err != nil {
			_, _ = conn.Write(resp.WriteError("ERR " + err.Error()))
			return
		}
		if _, err := conn.Write(out); err != nil {
			return
		}
	}
}
