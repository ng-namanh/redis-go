package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/codecrafters-io/redis-starter-go/internal/resp"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		v, err := resp.ReadValue(r)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Println("Error reading RESP: ", err.Error())
			return
		}

		out, err := DispatchCommand(v)
		if err != nil {
			_, _ = conn.Write(resp.AppendError(nil, "ERR "+err.Error()))
			return
		}
		if _, err := conn.Write(out); err != nil {
			return
		}
	}
}

func main() {
	fmt.Println("Logs from your program will appear here!")

	listener, err := net.Listen("tcp", "0.0.0.0:6379")

	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}
}
