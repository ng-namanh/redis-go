package redis_test

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ng-namanh/redis-go/internal/redis"
	"github.com/ng-namanh/redis-go/internal/resp"
)

func TestReplicaHandshake(t *testing.T) {
	// 1. Start a mock master server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	masterAddr := ln.Addr().String()
	host, port, _ := net.SplitHostPort(masterAddr)

	// 2. Configure replication settings
	redis.ResetForTesting()
	// We need to set these global variables in the commands package.
	// Since they are not re-exported by the shim yet, I'll add them if needed,
	// or just use the ones I already have if I can.
	// Wait, I should add SetHandshakeConfig to the redis shim.
	redis.SetHandshakeConfig(host, port, "6381")

	// 3. Start a goroutine to act as the master
	done := make(chan bool)
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)

		// Expected stages
		stages := []struct {
			expectedCmd string
			response    string
		}{
			{"PING", "+PONG\r\n"},
			{"REPLCONF", "+OK\r\n"},
			{"REPLCONF", "+OK\r\n"},
			{"PSYNC", "+FULLRESYNC mockid 0\r\n"},
		}

		for _, stage := range stages {
			v, err := resp.ReadValue(r)
			if err != nil {
				t.Errorf("mock master: error reading: %v", err)
				return
			}
			cmd, _, _ := resp.ParseCommand(v)
			if strings.ToUpper(cmd) != stage.expectedCmd {
				t.Errorf("mock master: expected command %s, got %s", stage.expectedCmd, cmd)
				return
			}
			conn.Write([]byte(stage.response))
		}
	}()

	// 4. Trigger the handshake
	// We need to call StartReplicaHandshake. Let's re-export it in the shim.
	redis.TriggerHandshake()

	// Wait for the mock master to finish or timeout
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("handshake timed out")
	}
}
