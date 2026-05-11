package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ng-namanh/redis-go/internal/resp"
)

func main() {
	host := flag.String("h", "127.0.0.1", "Server hostname")
	port := flag.Int("p", 6379, "Server port")
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("Could not connect to Redis at %s: %v\n", addr, err)
		os.Exit(1)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("%s> ", addr)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			fmt.Printf("%s> ", addr)
			continue
		}

		if line == "quit" || line == "exit" {
			break
		}

		args := splitArguments(line)
		if len(args) == 0 {
			fmt.Printf("%s> ", addr)
			continue
		}

		respArgs := make([]resp.RESP, len(args))
		for i, arg := range args {
			respArgs[i] = resp.RESP{Type: resp.BulkString, Str: arg}
		}

		cmdArray := resp.WriteArray(respArgs)
		_, err := conn.Write(cmdArray)
		if err != nil {
			fmt.Printf("Error writing to server: %v\n", err)
			break
		}

		val, err := resp.ReadValue(reader)
		if err != nil {
			fmt.Printf("Error reading from server: %v\n", err)
			break
		}

		printRESP(val, 0, false)
		fmt.Printf("%s> ", addr)
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

func splitArguments(line string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, r := range line {
		switch {
		case (r == '"' || r == '\'') && !inQuotes:
			inQuotes = true
			quoteChar = r
		case r == quoteChar && inQuotes:
			inQuotes = false
			quoteChar = rune(0)
		case r == ' ' && !inQuotes:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func printRESP(v resp.RESP, indent int, skipPrefix bool) {
	prefix := ""
	if !skipPrefix {
		prefix = strings.Repeat("  ", indent)
	}

	switch v.Type {
	case resp.SimpleString:
		fmt.Printf("%s%s\n", prefix, v.Str)
	case resp.Error:
		fmt.Printf("%s(error) %s\n", prefix, v.Err)
	case resp.Integer:
		fmt.Printf("%s(integer) %d\n", prefix, v.Int)
	case resp.BulkString:
		if v.Null {
			fmt.Printf("%s(nil)\n", prefix)
		} else {
			fmt.Printf("%s\"%s\"\n", prefix, v.Str)
		}
	case resp.Array:
		if v.Null {
			fmt.Printf("%s(nil)\n", prefix)
		} else {
			if len(v.Elems) == 0 {
				fmt.Printf("%s(empty array)\n", prefix)
				return
			}
			for i, elem := range v.Elems {
				itemPrefix := fmt.Sprintf("%s%d) ", prefix, i+1)
				fmt.Print(itemPrefix)
				// Align nested elements
				newIndent := indent + (len(itemPrefix)-len(prefix))/2
				printRESP(elem, newIndent, true)
			}
		}
	}
}
