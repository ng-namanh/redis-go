package redis_test

import (
	"bytes"
	"testing"

	"github.com/ng-namanh/redis-go/internal/redis"
	"github.com/ng-namanh/redis-go/internal/resp"
)

func TestECHO(t *testing.T) {
	t.Run("returns bulk string for single argument", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("ECHO", "Hello"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteBulkString("Hello")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("errors when no arguments", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("ECHO")); err == nil {
			t.Fatal("want error")
		}
	})
}
