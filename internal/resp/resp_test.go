package resp

import (
	"bufio"
	"strings"
	"testing"
)

func TestReadValue(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Value
		wantErr bool
	}{
		{
			name:  "simple string",
			input: "+OK\r\n",
			want:  Value{Type: SimpleString, Str: "OK"},
		},
		{
			name:  "error",
			input: "-ERR bad\r\n",
			want:  Value{Type: Error, Err: "ERR bad"},
		},
		{
			name:  "integer",
			input: ":42\r\n",
			want:  Value{Type: Integer, Int: 42},
		},
		{
			name:  "negative integer",
			input: ":-99\r\n",
			want:  Value{Type: Integer, Int: -99},
		},
		{
			name:  "bulk string",
			input: "$5\r\nhello\r\n",
			want:  Value{Type: BulkString, Str: "hello"},
		},
		{
			name:  "empty bulk",
			input: "$0\r\n\r\n",
			want:  Value{Type: BulkString, Str: ""},
		},
		{
			name:  "null bulk",
			input: "$-1\r\n",
			want:  Value{Type: BulkString, Null: true},
		},
		{
			name:  "null array",
			input: "*-1\r\n",
			want:  Value{Type: Array, Null: true},
		},
		{
			name:  "empty array",
			input: "*0\r\n",
			want:  Value{Type: Array, Elems: []Value{}},
		},
		{
			name:  "array of bulks",
			input: "*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n",
			want: Value{
				Type: Array,
				Elems: []Value{
					{Type: BulkString, Str: "foo"},
					{Type: BulkString, Str: "bar"},
				},
			},
		},
		{
			name:  "nested array",
			input: "*1\r\n*1\r\n$4\r\nping\r\n",
			want: Value{
				Type: Array,
				Elems: []Value{
					{
						Type: Array,
						Elems: []Value{
							{Type: BulkString, Str: "ping"},
						},
					},
				},
			},
		},
		{
			name:    "bad type byte",
			input:   "%x\r\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(tt.input))
			got, err := ReadValue(r)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !valueEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func valueEqual(a, b Value) bool {
	if a.Type != b.Type || a.Str != b.Str || a.Int != b.Int || a.Err != b.Err || a.Null != b.Null {
		return false
	}
	if len(a.Elems) != len(b.Elems) {
		return false
	}
	for i := range a.Elems {
		if !valueEqual(a.Elems[i], b.Elems[i]) {
			return false
		}
	}
	return true
}

func TestReadValue_pipelining(t *testing.T) {
	in := "+PONG\r\n+PONG\r\n"
	r := bufio.NewReader(strings.NewReader(in))
	v1, err := ReadValue(r)
	if err != nil {
		t.Fatal(err)
	}
	if v1.Type != SimpleString || v1.Str != "PONG" {
		t.Fatalf("first: %+v", v1)
	}
	v2, err := ReadValue(r)
	if err != nil {
		t.Fatal(err)
	}
	if v2.Type != SimpleString || v2.Str != "PONG" {
		t.Fatalf("second: %+v", v2)
	}
}

func TestParseCommand(t *testing.T) {
	v := Value{
		Type: Array,
		Elems: []Value{
			{Type: BulkString, Str: "ping"},
		},
	}
	cmd, args, err := ParseCommand(v)
	if err != nil || cmd != "PING" || len(args) != 0 {
		t.Fatalf("cmd=%q args=%v err=%v", cmd, args, err)
	}

	v2 := Value{
		Type: Array,
		Elems: []Value{
			{Type: BulkString, Str: "echo"},
			{Type: BulkString, Str: "Hi"},
		},
	}
	cmd, args, err = ParseCommand(v2)
	if err != nil || cmd != "ECHO" || len(args) != 1 || args[0] != "Hi" {
		t.Fatalf("cmd=%q args=%v err=%v", cmd, args, err)
	}
}

func TestAppendEncoders_roundTrip(t *testing.T) {
	if string(AppendSimpleString(nil, "OK")) != "+OK\r\n" {
		t.Fatal("simple string encoding")
	}
	if string(AppendInteger(nil, 7)) != ":7\r\n" {
		t.Fatal("integer encoding")
	}
	if string(AppendBulkString(nil, "ab")) != "$2\r\nab\r\n" {
		t.Fatal("bulk encoding")
	}
	if string(AppendError(nil, "ERR oops")) != "-ERR oops\r\n" {
		t.Fatal("error encoding")
	}
}
