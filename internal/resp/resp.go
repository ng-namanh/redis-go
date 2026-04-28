package resp

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Type byte

const (
	SimpleString Type = iota
	Error
	Integer
	BulkString
	Array
)

type Value struct {
	Type  Type
	Str   string // simple string or bulk body
	Int   int64
	Err   string // when Type is Error
	Elems []Value
	Null  bool // null bulk string or null array
}

func ReadValue(r *bufio.Reader) (Value, error) {
	b, err := r.ReadByte()
	if err != nil {
		return Value{}, err
	}
	switch b {
	case '+':
		line, err := readLineCRLF(r)
		if err != nil {
			return Value{}, err
		}
		return Value{Type: SimpleString, Str: line}, nil
	case '-':
		line, err := readLineCRLF(r)
		if err != nil {
			return Value{}, err
		}
		return Value{Type: Error, Err: line}, nil
	case ':':
		line, err := readLineCRLF(r)
		if err != nil {
			return Value{}, err
		}
		n, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return Value{}, fmt.Errorf("invalid integer: %w", err)
		}
		return Value{Type: Integer, Int: n}, nil
	case '$':
		line, err := readLineCRLF(r)
		if err != nil {
			return Value{}, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return Value{}, fmt.Errorf("invalid bulk length: %w", err)
		}
		if n == -1 {
			return Value{Type: BulkString, Null: true}, nil
		}
		if n < -1 {
			return Value{}, fmt.Errorf("invalid bulk length %d", n)
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(r, body); err != nil {
			return Value{}, err
		}
		if err := readExactCRLF(r); err != nil {
			return Value{}, err
		}
		return Value{Type: BulkString, Str: string(body)}, nil
	case '*':
		line, err := readLineCRLF(r)
		if err != nil {
			return Value{}, err
		}
		cnt, err := strconv.Atoi(line)
		if err != nil {
			return Value{}, fmt.Errorf("invalid array length: %w", err)
		}
		if cnt == -1 {
			return Value{Type: Array, Null: true}, nil
		}
		if cnt < -1 {
			return Value{}, fmt.Errorf("invalid array length %d", cnt)
		}
		elems := make([]Value, 0, cnt)
		for i := 0; i < cnt; i++ {
			v, err := ReadValue(r)
			if err != nil {
				return Value{}, err
			}
			elems = append(elems, v)
		}
		return Value{Type: Array, Elems: elems}, nil
	default:
		return Value{}, fmt.Errorf("unknown RESP type byte %q", b)
	}
}

// Read one RESP line that must end with `\r\n` and returns the content without that terminator
func readLineCRLF(r *bufio.Reader) (string, error) {
	line, err := r.ReadBytes('\n')

	if err != nil {
		return "", err
	}

	// checks that the byte before `\n` is `\r`
	if len(line) < 2 || line[len(line)-2] != '\r' {
		return "", fmt.Errorf("invalid line ending")
	}

	// removes the trailing `\r\n` and returns only the payload
	return string(line[:len(line)-2]), nil
}

func readExactCRLF(r *bufio.Reader) error {
	cr, err := r.ReadByte()
	if err != nil {
		return err
	}
	if cr != '\r' {
		return fmt.Errorf("expected \\r after bulk data")
	}
	lf, err := r.ReadByte()
	if err != nil {
		return err
	}
	if lf != '\n' {
		return fmt.Errorf("expected \\n after bulk data")
	}
	return nil
}

// AppendSimpleString appends a RESP simple string (+...\r\n).
func AppendSimpleString(buf []byte, s string) []byte {
	buf = append(buf, '+')
	buf = append(buf, s...)
	return append(buf, '\r', '\n')
}

// AppendInteger appends a RESP integer (:\r\n).
func AppendInteger(buf []byte, n int64) []byte {
	buf = append(buf, ':')
	buf = strconv.AppendInt(buf, n, 10)
	return append(buf, '\r', '\n')
}

// AppendBulkString appends a RESP bulk string ($<len>\r\n...\r\n).
func AppendBulkString(buf []byte, s string) []byte {
	buf = append(buf, '$')
	buf = strconv.AppendInt(buf, int64(len(s)), 10)
	buf = append(buf, '\r', '\n')
	buf = append(buf, s...)
	return append(buf, '\r', '\n')
}

// AppendError appends a RESP error (-...\r\n).
func AppendError(buf []byte, s string) []byte {
	buf = append(buf, '-')
	buf = append(buf, s...)
	return append(buf, '\r', '\n')
}

// ParseCommand treats v as a request array: [cmd bulk, arg bulk, ...].
// Command name is uppercased; arguments keep original case.
func ParseCommand(v Value) (cmd string, args []string, err error) {
	if v.Type != Array || v.Null {
		return "", nil, fmt.Errorf("expected non-null array")
	}
	if len(v.Elems) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}
	first := v.Elems[0]
	if first.Type != BulkString || first.Null {
		return "", nil, fmt.Errorf("command must be a bulk string")
	}
	cmd = strings.ToUpper(first.Str)
	for _, e := range v.Elems[1:] {
		if e.Type != BulkString || e.Null {
			return "", nil, fmt.Errorf("arguments must be bulk strings")
		}
		args = append(args, e.Str)
	}
	return cmd, args, nil
}
