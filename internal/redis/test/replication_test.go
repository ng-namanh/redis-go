package redis_test

import (
	"bytes"
	"testing"

	"github.com/ng-namanh/redis-go/internal/redis"
)

func TestINFO(t *testing.T) {
	redis.ResetForTesting()

	res, err := redis.DispatchCommand(req("INFO"))
	if err != nil {
		t.Fatalf("INFO failed: %v", err)
	}

	// Response should be a bulk string containing role:master
	want := "role:master\r\n"
	got := mustReadBulkString(t, res)
	if !bytes.Contains([]byte(got), []byte(want)) {
		t.Fatalf("expected role:master, got %q", got)
	}
}

func TestINFOSlave(t *testing.T) {
	redis.ResetForTesting()
	redis.SetRole("slave")

	res, err := redis.DispatchCommand(req("INFO"))
	if err != nil {
		t.Fatalf("INFO failed: %v", err)
	}

	want := "role:slave\r\n"
	got := mustReadBulkString(t, res)
	if !bytes.Contains([]byte(got), []byte(want)) {
		t.Fatalf("expected role:slave, got %q", got)
	}
}
