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

	got := mustReadBulkString(t, res)
	if !bytes.Contains([]byte(got), []byte("role:master\r\n")) {
		t.Fatalf("expected role:master, got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("master_replid:")) {
		t.Fatalf("expected master_replid, got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("master_repl_offset:0\r\n")) {
		t.Fatalf("expected master_repl_offset:0, got %q", got)
	}
}

func TestINFOSlave(t *testing.T) {
	redis.ResetForTesting()
	redis.SetRole("slave")

	res, err := redis.DispatchCommand(req("INFO"))
	if err != nil {
		t.Fatalf("INFO failed: %v", err)
	}

	got := mustReadBulkString(t, res)
	if !bytes.Contains([]byte(got), []byte("role:slave\r\n")) {
		t.Fatalf("expected role:slave, got %q", got)
	}
	if !bytes.Contains([]byte(got), []byte("master_replid:")) {
		t.Fatalf("expected master_replid, got %q", got)
	}
}
