package resp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Type byte

const CRLF = "\r\n"

const (
	SimpleString Type = iota
	Error
	Integer
	BulkString
	Array
)

type RESP struct {
	Type  Type
	Str   string // simple string or bulk body
	Int   int64
	Err   string // when Type is Error
	Elems []RESP
	Null  bool // null bulk string or null array
}

// ReadValue parses a single RESP value from r.
//
// Format recap (RESP2):
//
//	+<text>\r\n                 Simple String
//	-<text>\r\n                 Error
//	:<number>\r\n               Integer
//	$<len>\r\n<data>\r\n        Bulk String (len == -1 means null bulk string)
//	*<count>\r\n<elem>...       Array (count == -1 means null array)
//
// Implementation notes:
//   - We read the first type byte, then read the CRLF-terminated "header line"
//     using readLineCRLF() for all types that have one.
//   - Bulk strings then read exactly <len> bytes and require a trailing \r\n
//     (readExactCRLF) to avoid accepting truncated/overlong bodies.
//   - Arrays recursively call ReadValue() <count> times to parse each element.
func ReadValue(r *bufio.Reader) (RESP, error) {
	b, err := r.ReadByte()
	if err != nil {
		return RESP{}, err
	}
	switch b {
	case '+':
		line, err := readLineCRLF(r)
		if err != nil {
			return RESP{}, err
		}
		return RESP{Type: SimpleString, Str: line}, nil
	case '-':
		line, err := readLineCRLF(r)
		if err != nil {
			return RESP{}, err
		}
		return RESP{Type: Error, Err: line}, nil
	case ':':
		line, err := readLineCRLF(r)
		if err != nil {
			return RESP{}, err
		}
		n, err := strconv.ParseInt(line, 10, 64)
		if err != nil {
			return RESP{}, fmt.Errorf("invalid integer: %w", err)
		}
		return RESP{Type: Integer, Int: n}, nil
	case '$':
		line, err := readLineCRLF(r)
		if err != nil {
			return RESP{}, err
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			return RESP{}, fmt.Errorf("invalid bulk length: %w", err)
		}

		// if the bulk length is -1, it means the bulk string is null
		if n == -1 {
			return RESP{Type: BulkString, Null: true}, nil
		}

		// if the bulk length is less than -1, it means the bulk string is invalid
		if n < -1 {
			return RESP{}, fmt.Errorf("invalid bulk length %d", n)
		}

		body := make([]byte, n)

		if _, err := io.ReadFull(r, body); err != nil {
			return RESP{}, err
		}

		if err := readExactCRLF(r); err != nil {
			return RESP{}, err
		}

		return RESP{Type: BulkString, Str: string(body)}, nil
	case '*':
		line, err := readLineCRLF(r)
		if err != nil {
			return RESP{}, err
		}
		cnt, err := strconv.Atoi(line)

		if err != nil {
			return RESP{}, fmt.Errorf("invalid array length: %w", err)
		}

		if cnt == -1 {
			return RESP{Type: Array, Null: true}, nil
		}

		if cnt < -1 {
			return RESP{}, fmt.Errorf("invalid array length %d", cnt)
		}

		elems := make([]RESP, 0, cnt)
		for i := 0; i < cnt; i++ {
			v, err := ReadValue(r)
			if err != nil {
				return RESP{}, err
			}
			elems = append(elems, v)
		}
		return RESP{Type: Array, Elems: elems}, nil
	default:
		return RESP{}, fmt.Errorf("unknown RESP type byte %q", b)
	}
}

// readLineCRLF reads bytes up to '\n' and enforces a "\r\n" line ending.
// It returns only the payload (without the trailing "\r\n").
//
// Example: input "PING\r\n" -> returns "PING".
// If the line ends with "\n" without a preceding "\r", it returns "invalid line ending".
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

// readExactCRLF consumes the exact 2-byte terminator that RESP requires immediately after a bulk string’s n payload bytes: \r\n.
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

func WriteSimpleString(s string) []byte {
	return []byte("+" + s + CRLF)
}

func WriteInteger(n int64) []byte {
	return []byte(":" + strconv.FormatInt(n, 10) + CRLF)
}

func WriteBulkString(s string) []byte {
	return []byte("$" + strconv.FormatInt(int64(len(s)), 10) + CRLF + s + CRLF)
}

func WriteError(s string) []byte {
	return []byte("-" + s + CRLF)
}

func encodeRESP(v RESP) []byte {
	switch v.Type {
	case SimpleString:
		return WriteSimpleString(v.Str)
	case Error:
		return WriteError(v.Err)
	case Integer:
		return WriteInteger(v.Int)
	case BulkString:
		if v.Null {
			return []byte("$-1" + CRLF)
		}
		return WriteBulkString(v.Str)
	case Array:
		if v.Null {
			return []byte("*-1" + CRLF)
		}
		var b bytes.Buffer
		b.WriteByte('*')
		b.WriteString(strconv.Itoa(len(v.Elems)))
		b.WriteString(CRLF)
		for _, e := range v.Elems {
			b.Write(encodeRESP(e))
		}
		return b.Bytes()
	default:
		return WriteBulkString(v.Str)
	}
}

// WriteArray encodes elems as *<n>\r\n followed by each element as a full RESP value.
func WriteArray(elems []RESP) []byte {
	if elems == nil {
		return []byte("*-1" + CRLF)
	}
	var b bytes.Buffer
	b.WriteByte('*')
	b.WriteString(strconv.FormatInt(int64(len(elems)), 10))
	b.WriteString(CRLF)
	for _, e := range elems {
		b.Write(encodeRESP(e))
	}
	return b.Bytes()
}

// ParseCommand treats v as a request array: [cmd bulk, arg bulk, ...].
// Command name is uppercased; arguments keep original case.
func ParseCommand(v RESP) (cmd string, args []string, err error) {
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
