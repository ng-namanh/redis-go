package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/ng-namanh/redis-go/internal/commands"
	"github.com/ng-namanh/redis-go/internal/server"
)

func main() {
	port := flag.Int("port", 6379, "The port on which the server will listen for incoming connections.")
	replicaof := flag.String("replicaof", "", "Create a replica of another Redis server. Expects 'master_host master_port'.")
	flag.Parse()

	commands.ServerPort = strconv.Itoa(*port)

	if *replicaof != "" {
		parts := strings.Fields(*replicaof)
		if len(parts) == 2 {
			commands.Role = "slave"
			commands.MasterHost = parts[0]
			commands.MasterPort = parts[1]
			go commands.StartReplicaHandshake()
		}
	}

	addr := fmt.Sprintf("0.0.0.0:%d", *port)
	fmt.Printf("Redis server started. Listening on %s\n", addr)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Printf("Failed to bind to port %d\n", *port)
		os.Exit(1)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go server.Handle(conn)
	}
}
