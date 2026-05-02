package redis_test

import (
	"bytes"
	"testing"

	"github.com/ng-namanh/redis-go/internal/redis"
	"github.com/ng-namanh/redis-go/internal/resp"
)

// TestXADD specifies behavior from docs/xadd.md (explicit ID, bulk string reply; TYPE is stream).
func TestXADD(t *testing.T) {
	t.Run("errors when fewer than key id and one field pair", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.XADD([]string{"sk"}); err == nil {
			t.Fatal("want error")
		}
		redis.ResetForTesting()
		if _, err := redis.XADD([]string{"sk", "0-1"}); err == nil {
			t.Fatal("want error")
		}
		redis.ResetForTesting()
		if _, err := redis.XADD([]string{"sk", "0-1", "onlyfield"}); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("returns id as bulk string (Codecrafters example 0-1)", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("XADD", "stream_key", "0-1", "foo", "bar"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteBulkString("0-1")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q (e.g. $3\\r\\n0-1\\r\\n)", got, want)
		}
	})

	t.Run("returns id as bulk string for multi-field entry", func(t *testing.T) {
		redis.ResetForTesting()
		id := "1526919030474-0"
		got, err := redis.DispatchCommand(req("XADD", "weather", id, "temperature", "36", "humidity", "95"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteBulkString(id)) {
			t.Fatalf("got %q, want bulk %q", got, id)
		}
	})

	t.Run("TYPE stream_key is stream after XADD", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "stream_key", "0-1", "foo", "bar"))
		got, err := redis.DispatchCommand(req("TYPE", "stream_key"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteSimpleString("stream")) {
			t.Fatalf("got %q, want +stream\\r\\n", got)
		}
	})

	t.Run("creates stream when key did not exist", func(t *testing.T) {
		redis.ResetForTesting()
		_, err := redis.DispatchCommand(req("XADD", "new_stream", "1-0", "k", "v"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := redis.TYPE([]string{"new_stream"})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteSimpleString("stream")) {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("accepts strictly increasing IDs 1-1 then 1-2", func(t *testing.T) {
		redis.ResetForTesting()
		got1, err := redis.DispatchCommand(req("XADD", "stream_key", "1-1", "foo", "bar"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got1, resp.WriteBulkString("1-1")) {
			t.Fatalf("first XADD: got %q", got1)
		}
		got2, err := redis.DispatchCommand(req("XADD", "stream_key", "1-2", "bar", "baz"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got2, resp.WriteBulkString("1-2")) {
			t.Fatalf("second XADD: got %q", got2)
		}
	})

	t.Run("rejects ID equal to top entry", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "stream_key", "1-1", "foo", "bar"))
		mustDispatch(t, req("XADD", "stream_key", "1-2", "bar", "baz"))
		got, err := redis.DispatchCommand(req("XADD", "stream_key", "1-2", "baz", "foo"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteError("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("rejects ID with smaller time than top even if sequence is larger", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "stream_key", "1-1", "foo", "bar"))
		got, err := redis.DispatchCommand(req("XADD", "stream_key", "0-3", "baz", "foo"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteError("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("rejects 0-0 on empty stream", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("XADD", "stream_key", "0-0", "baz", "foo"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteError("ERR The ID specified in XADD must be greater than 0-0")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("rejects 0-0 when stream already has entries", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "stream_key", "1-1", "foo", "bar"))
		got, err := redis.DispatchCommand(req("XADD", "stream_key", "0-0", "baz", "foo"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteError("ERR The ID specified in XADD must be greater than 0-0")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("asterisk auto id returns bulk string ms-seq", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("XADD", "stream_key", "*", "foo", "bar"))
		if err != nil {
			t.Fatal(err)
		}
		id := mustReadBulkString(t, got)
		parsed, ok := redis.ParseStreamID(id)
		if !ok {
			t.Fatalf("not a valid stream id: %q", id)
		}
		_ = parsed
		if !bytes.Equal(got, resp.WriteBulkString(id)) {
			t.Fatalf("wire encoding mismatch")
		}
	})

	t.Run("consecutive asterisk XADD ids are strictly increasing", func(t *testing.T) {
		redis.ResetForTesting()
		got1, err := redis.DispatchCommand(req("XADD", "stream_key", "*", "a", "1"))
		if err != nil {
			t.Fatal(err)
		}
		got2, err := redis.DispatchCommand(req("XADD", "stream_key", "*", "b", "2"))
		if err != nil {
			t.Fatal(err)
		}
		id1, ok1 := redis.ParseStreamID(mustReadBulkString(t, got1))
		id2, ok2 := redis.ParseStreamID(mustReadBulkString(t, got2))
		if !ok1 || !ok2 {
			t.Fatal("parse id")
		}
		if !id2.GreaterThan(id1) {
			t.Fatalf("want id2 > id1, got %v and %v", id1, id2)
		}
	})

	t.Run("asterisk after explicit id generates id greater than last", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "stream_key", "9999999999999-42", "k", "v"))
		got, err := redis.DispatchCommand(req("XADD", "stream_key", "*", "foo", "bar"))
		if err != nil {
			t.Fatal(err)
		}
		autoID, ok := redis.ParseStreamID(mustReadBulkString(t, got))
		lastID, ok2 := redis.ParseStreamID("9999999999999-42")
		if !ok || !ok2 {
			t.Fatal("parse")
		}
		if !autoID.GreaterThan(lastID) {
			t.Fatalf("auto id %v not greater than last %v", autoID, lastID)
		}
	})

	t.Run("partial seq 1-* first entry is 1-0", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("XADD", "some_key", "1-*", "foo", "bar"))
		if err != nil {
			t.Fatal(err)
		}
		if mustReadBulkString(t, got) != "1-0" {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("partial seq 1-* second entry increments sequence", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "some_key", "1-*", "foo", "bar"))
		got, err := redis.DispatchCommand(req("XADD", "some_key", "1-*", "bar", "baz"))
		if err != nil {
			t.Fatal(err)
		}
		if mustReadBulkString(t, got) != "1-1" {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("partial seq rejects ms smaller than top entry", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "some_key", "2-0", "k", "v"))
		got, err := redis.DispatchCommand(req("XADD", "some_key", "1-*", "a", "b"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteError("ERR The ID specified in XADD is equal or smaller than the target stream top item")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("partial seq 0-* on empty stream is 0-1", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("XADD", "z", "0-*", "x", "y"))
		if err != nil {
			t.Fatal(err)
		}
		if mustReadBulkString(t, got) != "0-1" {
			t.Fatalf("got %s", got)
		}
	})
}
