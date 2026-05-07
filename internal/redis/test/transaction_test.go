package redis_test

import (
	"bytes"
	"testing"

	"github.com/ng-namanh/redis-go/internal/client"
	"github.com/ng-namanh/redis-go/internal/redis"
	"github.com/ng-namanh/redis-go/internal/resp"
)

func TestTransaction(t *testing.T) {
	t.Run("basic MULTI/EXEC", func(t *testing.T) {
		redis.ResetForTesting()
		c := client.NewClient()

		// Start transaction
		res, err := c.DispatchCommand(req("MULTI"))
		if err != nil || !bytes.Equal(res, resp.WriteSimpleString("OK")) {
			t.Fatalf("MULTI failed: %v, %q", err, res)
		}

		// Queue commands
		res, err = c.DispatchCommand(req("SET", "foo", "bar"))
		if err != nil || !bytes.Equal(res, resp.WriteSimpleString("QUEUED")) {
			t.Fatalf("SET failed: %v, %q", err, res)
		}

		res, err = c.DispatchCommand(req("GET", "foo"))
		if err != nil || !bytes.Equal(res, resp.WriteSimpleString("QUEUED")) {
			t.Fatalf("GET failed: %v, %q", err, res)
		}

		// Execute
		res, err = c.DispatchCommand(req("EXEC"))
		if err != nil {
			t.Fatalf("EXEC failed: %v", err)
		}

		want := resp.WriteArray([]resp.RESP{
			{Type: resp.SimpleString, Str: "OK"},
			{Type: resp.BulkString, Str: "bar"},
		})
		if !bytes.Equal(res, want) {
			t.Fatalf("EXEC result mismatch: got %q, want %q", res, want)
		}
	})

	t.Run("DISCARD works", func(t *testing.T) {
		redis.ResetForTesting()
		c := client.NewClient()

		c.DispatchCommand(req("MULTI"))
		c.DispatchCommand(req("SET", "foo", "bar"))

		res, err := c.DispatchCommand(req("DISCARD"))
		if err != nil || !bytes.Equal(res, resp.WriteSimpleString("OK")) {
			t.Fatalf("DISCARD failed: %v, %q", err, res)
		}

		// Check that SET was NOT executed
		res, err = c.DispatchCommand(req("GET", "foo"))
		if err != nil || !bytes.Contains(res, []byte("$-1")) {
			t.Fatalf("GET should be nil: %q", res)
		}
	})

	t.Run("nested MULTI errors", func(t *testing.T) {
		redis.ResetForTesting()
		c := client.NewClient()

		c.DispatchCommand(req("MULTI"))
		_, err := c.DispatchCommand(req("MULTI"))
		if err == nil {
			t.Fatal("expected error for nested MULTI")
		}
	})

	t.Run("EXEC without MULTI errors", func(t *testing.T) {
		redis.ResetForTesting()
		c := client.NewClient()

		_, err := c.DispatchCommand(req("EXEC"))
		if err == nil {
			t.Fatal("expected error for EXEC without MULTI")
		}
	})

	t.Run("DISCARD without MULTI errors", func(t *testing.T) {
		redis.ResetForTesting()
		c := client.NewClient()

		_, err := c.DispatchCommand(req("DISCARD"))
		if err == nil {
			t.Fatal("expected error for DISCARD without MULTI")
		}
	})

	t.Run("empty EXEC returns empty array", func(t *testing.T) {
		redis.ResetForTesting()
		c := client.NewClient()

		c.DispatchCommand(req("MULTI"))
		res, err := c.DispatchCommand(req("EXEC"))
		if err != nil {
			t.Fatalf("EXEC failed: %v", err)
		}
		if !bytes.Equal(res, resp.WriteArray(nil)) {
			t.Fatalf("expected empty array, got %q", res)
		}
	})

	t.Run("command failure inside EXEC", func(t *testing.T) {
		redis.ResetForTesting()
		c := client.NewClient()

		c.DispatchCommand(req("MULTI"))
		// SET a string
		c.DispatchCommand(req("SET", "foo", "bar"))
		// INCR on a string should fail
		c.DispatchCommand(req("INCR", "foo"))
		// GET should still work later in the transaction
		c.DispatchCommand(req("GET", "foo"))

		res, err := c.DispatchCommand(req("EXEC"))
		if err != nil {
			t.Fatalf("EXEC failed: %v", err)
		}

		// We expect: [OK, Error, "bar"]
		want := resp.WriteArray([]resp.RESP{
			{Type: resp.SimpleString, Str: "OK"},
			{Type: resp.Error, Err: "ERR WRONGTYPE Operation against a key holding the wrong kind of value"},
			{Type: resp.BulkString, Str: "bar"},
		})
		if !bytes.Equal(res, want) {
			t.Fatalf("EXEC result mismatch: got %q, want %q", res, want)
		}
	})

	t.Run("independent transactions for multiple clients", func(t *testing.T) {
		redis.ResetForTesting()
		c1 := client.NewClient()
		c2 := client.NewClient()

		c1.DispatchCommand(req("MULTI"))
		c1.DispatchCommand(req("SET", "a", "1"))

		c2.DispatchCommand(req("MULTI"))
		c2.DispatchCommand(req("SET", "a", "2"))

		// c1 executes first
		c1.DispatchCommand(req("EXEC"))

		// check value
		res, _ := redis.DispatchCommand(req("GET", "a"))
		if !bytes.Contains(res, []byte("1")) {
			t.Fatalf("expected 1, got %q", res)
		}

		// c2 executes
		c2.DispatchCommand(req("EXEC"))

		// check value again
		res, _ = redis.DispatchCommand(req("GET", "a"))
		if !bytes.Contains(res, []byte("2")) {
			t.Fatalf("expected 2, got %q", res)
		}
	})
}
