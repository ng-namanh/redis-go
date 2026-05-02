package redis_test

import (
	"bufio"
	"bytes"
	"testing"

	"github.com/ng-namanh/redis-go/internal/redis"
	"github.com/ng-namanh/redis-go/internal/resp"
)

func req(parts ...string) resp.RESP {
	elems := make([]resp.RESP, len(parts))
	for i, p := range parts {
		elems[i] = resp.RESP{Type: resp.BulkString, Str: p}
	}
	return resp.RESP{Type: resp.Array, Elems: elems}
}

func mustDispatch(t *testing.T, v resp.RESP) []byte {
	t.Helper()
	b, err := redis.DispatchCommand(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustReadBulkString(t *testing.T, wire []byte) string {
	t.Helper()
	v, err := resp.ReadValue(bufio.NewReader(bytes.NewReader(wire)))
	if err != nil {
		t.Fatal(err)
	}
	if v.Type != resp.BulkString || v.Null {
		t.Fatalf("want bulk string, got %+v", v)
	}
	return v.Str
}
