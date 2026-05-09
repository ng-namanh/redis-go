package commands

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strconv"
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
	fullResync := resp.WriteSimpleString(fmt.Sprintf("FULLRESYNC %s %d", MasterReplid, MasterReplOffset))

	rdbData, _ := base64.StdEncoding.DecodeString(emptyRDBBase64)
	rdbHeader := []byte(fmt.Sprintf("$%d\r\n", len(rdbData)))

	res := make([]byte, 0, len(fullResync)+len(rdbHeader)+len(rdbData))
	res = append(res, fullResync...)
	res = append(res, rdbHeader...)
	res = append(res, rdbData...)

	return res, nil
}

func StartReplicaHandshake() {
	addr := net.JoinHostPort(MasterHost, MasterPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("Failed to connect to master at %s: %v\n", addr, err)
		return
	}
	// Note: We keep this connection open to receive propagated commands.

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

	if _, err := resp.ReadValue(reader); err != nil {
		fmt.Printf("Handshake Stage 4 (PSYNC) FULLRESYNC failed: %v\n", err)
		return
	}

	b, err := reader.ReadByte()
	if err != nil || b != '$' {
		fmt.Printf("Handshake Stage 5 (RDB transfer) failed to read $: %v\n", err)
		return
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Handshake Stage 5 (RDB transfer) failed to read length: %v\n", err)
		return
	}
	length, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		fmt.Printf("Handshake Stage 5 (RDB transfer) invalid length: %v\n", err)
		return
	}

	rdbData := make([]byte, length)
	if _, err := io.ReadFull(reader, rdbData); err != nil {
		fmt.Printf("Handshake Stage 5 (RDB transfer) failed to read data: %v\n", err)
		return
	}

	go processMasterCommands(conn, reader)
}

func processMasterCommands(conn net.Conn, r *bufio.Reader) {
	defer conn.Close()
	for {
		v, err := resp.ReadValue(r)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Replica: error reading master command: %v\n", err)
			}
			return
		}

		cmd, args, err := resp.ParseCommand(v)
		if err != nil {
			fmt.Printf("Replica: error parsing master command: %v\n", err)
			continue
		}

		_, err = HandleCommand(strings.ToUpper(cmd), args)
		if err != nil {
			fmt.Printf("Replica: error executing master command %s: %v\n", cmd, err)
		}
	}
}

func sendCommand(conn net.Conn, args []string) error {
	elems := make([]resp.RESP, len(args))
	for i, arg := range args {
		elems[i] = resp.RESP{Type: resp.BulkString, Str: arg}
	}
	write := resp.WriteArray(elems)
	_, err := conn.Write(write)
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
