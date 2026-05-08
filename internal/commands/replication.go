package commands

import (
	"bufio"
	"fmt"
	"net"
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
		sb.WriteString(fmt.Sprintf("master_replid:%s\r\n", MasterReplid))
		sb.WriteString(fmt.Sprintf("master_repl_offset:%d\r\n", MasterReplOffset))
	}

	return resp.WriteBulkString(sb.String()), nil
}

func REPLCONF(args []string) ([]byte, error) {
	return resp.WriteSimpleString("OK"), nil
}

func PSYNC(args []string) ([]byte, error) {
	return resp.WriteSimpleString(fmt.Sprintf("FULLRESYNC %s %d", MasterReplid, MasterReplOffset)), nil
}

var dialer func(network, address string) (net.Conn, error) = net.Dial

func StartReplicaHandshake() {
	addr := net.JoinHostPort(MasterHost, MasterPort)
	conn, err := dialer("tcp", addr)
	if err != nil {
		fmt.Printf("Failed to connect to master at %s: %v\n", addr, err)
		return
	}
	// Note: In a real implementation, we would keep this connection open for replication.
	// For now, we follow the handshake stages.
	defer conn.Close()

	reader := bufio.NewReader(conn)

	if err := sendAndExpect(conn, reader, []string{"PING"}, "PONG"); err != nil {
		fmt.Printf("Handshake Stage 1 (PING) failed: %v\n", err)
		return
	}

	if err := sendAndExpect(conn, reader, []string{"REPLCONF", "listening-port", ServerPort}, "OK"); err != nil {
		fmt.Printf("Handshake Stage 2 (REPLCONF listening-port) failed: %v\n", err)
		return
	}

	if err := sendAndExpect(conn, reader, []string{"REPLCONF", "capa", "psync2"}, "OK"); err != nil {
		fmt.Printf("Handshake Stage 3 (REPLCONF capa) failed: %v\n", err)
		return
	}

	if err := sendCommand(conn, []string{"PSYNC", "?", "-1"}); err != nil {
		fmt.Printf("Handshake Stage 4 (PSYNC) failed: %v\n", err)
		return
	}
	// We don't necessarily need to check the response of PSYNC for this stage,
	// but we should read it to clear the buffer.
	_, _ = resp.ReadValue(reader)
}

func sendCommand(conn net.Conn, args []string) error {
	elems := make([]resp.RESP, len(args))
	for i, arg := range args {
		elems[i] = resp.RESP{Type: resp.BulkString, Str: arg}
	}
	wire := resp.WriteArray(elems)
	_, err := conn.Write(wire)
	return err
}

func sendAndExpect(conn net.Conn, reader *bufio.Reader, args []string, expected string) error {
	if err := sendCommand(conn, args); err != nil {
		return err
	}
	v, err := resp.ReadValue(reader)
	if err != nil {
		return err
	}

	actual := ""
	switch v.Type {
	case resp.SimpleString, resp.BulkString:
		actual = v.Str
	}

	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("expected %q, got %q", expected, actual)
	}
	return nil
}
