package redis_test

import (
	"bytes"
	"testing"

	"github.com/ng-namanh/redis-go/internal/redis"
	"github.com/ng-namanh/redis-go/internal/resp"
)

func TestSET(t *testing.T) {
	t.Run("errors when fewer than key and value", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("SET", "onlykey")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("stores value and replies OK", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("SET", "k", "v"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "+OK\r\n" {
			t.Fatalf("got %s", got)
		}
	})
}

func TestGET(t *testing.T) {
	t.Run("returns bulk string after SET", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("SET", "k", "v"))
		got, err := redis.DispatchCommand(req("GET", "k"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteBulkString("v")) {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("returns null bulk when key missing", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("GET", "missing"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "$-1\r\n" {
			t.Fatalf("got %q", got)
		}
	})
}
