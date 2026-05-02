package redis_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ng-namanh/redis-go/internal/redis"
	"github.com/ng-namanh/redis-go/internal/resp"
)

func TestRPUSH(t *testing.T) {
	t.Run("length accumulates across calls", func(t *testing.T) {
		redis.ResetForTesting()
		l1, err := redis.DispatchCommand(req("RPUSH", "fruit", "a"))
		if err != nil {
			t.Fatal(err)
		}
		if string(l1) != ":1\r\n" {
			t.Fatalf("first RPUSH got %s", l1)
		}
		l2, err := redis.DispatchCommand(req("RPUSH", "fruit", "b", "c"))
		if err != nil {
			t.Fatal(err)
		}
		if string(l2) != ":3\r\n" {
			t.Fatalf("got %s", l2)
		}
	})

	t.Run("errors when key but no values", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("RPUSH", "k")); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestLRANGE(t *testing.T) {
	t.Run("empty list returns zero-length array", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("LRANGE", "nix", "0", "-1"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "*0\r\n" {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("errors when fewer than three arguments", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("LRANGE", "k", "0")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("full range reflects LPUSH head order after RPUSH tail", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("RPUSH", "k", "tail"))
		mustDispatch(t, req("LPUSH", "k", "mid", "head"))
		got, err := redis.DispatchCommand(req("LRANGE", "k", "0", "-1"))
		if err != nil {
			t.Fatal(err)
		}
		want := "*3\r\n$4\r\nhead\r\n$3\r\nmid\r\n$4\r\ntail\r\n"
		if string(got) != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
}

func TestLLEN(t *testing.T) {
	t.Run("unknown key returns integer zero", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("LLEN", "noSuchKey"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":0\r\n" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("matches docs example after two LPUSH", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("LPUSH", "mylist", "World"))
		mustDispatch(t, req("LPUSH", "mylist", "Hello"))
		got, err := redis.DispatchCommand(req("LLEN", "mylist"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":2\r\n" {
			t.Fatalf("got %q want :2\\r\\n", got)
		}
	})

	t.Run("totals length after RPUSH with multiple values", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("RPUSH", "fruits", "a"))
		mustDispatch(t, req("RPUSH", "fruits", "b", "c"))
		got, err := redis.DispatchCommand(req("LLEN", "fruits"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":3\r\n" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("reply is RESP integer", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("RPUSH", "solo", "x"))
		got, err := redis.DispatchCommand(req("LLEN", "solo"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":1\r\n" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("exactly one key argument required", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("LLEN")); err == nil {
			t.Fatal("want error for arity 0")
		}
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("LLEN", "k", "extra")); err == nil {
			t.Fatal("want error for extra arguments")
		}
	})

	t.Run("WRONGTYPE when key is a string from SET", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("SET", "k", "v"))
		_, err := redis.DispatchCommand(req("LLEN", "k"))
		if err == nil {
			t.Fatal("want error")
		}
		if !strings.Contains(err.Error(), "WRONGTYPE") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestLPOP(t *testing.T) {
	t.Run("returns null bulk reply when key does not exist", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("LPOP", "missing"))
		if err != nil {
			t.Fatal(err)
		}
		want := "$-1\r\n"
		if string(got) != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("matches docs example RPUSH LPOP LPOP LRANGE", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("RPUSH", "mylist", "one", "two", "three", "four", "five"))

		got, err := redis.DispatchCommand(req("LPOP", "mylist"))
		if err != nil {
			t.Fatal(err)
		}
		wantOne := resp.WriteBulkString("one")
		if !bytes.Equal(got, wantOne) {
			t.Fatalf("first LPOP got %q want %q", got, wantOne)
		}

		got, err = redis.DispatchCommand(req("LPOP", "mylist", "2"))
		if err != nil {
			t.Fatal(err)
		}
		wantTwoThree := "*2\r\n$3\r\ntwo\r\n$5\r\nthree\r\n"
		if string(got) != wantTwoThree {
			t.Fatalf("LPOP with count:\ngot %q\nwant %q", got, wantTwoThree)
		}

		got, err = redis.DispatchCommand(req("LRANGE", "mylist", "0", "-1"))
		if err != nil {
			t.Fatal(err)
		}
		wantRange := "*2\r\n$4\r\nfour\r\n$4\r\nfive\r\n"
		if string(got) != wantRange {
			t.Fatalf("LRANGE after pops:\ngot %q\nwant %q", got, wantRange)
		}
	})

	t.Run("errors when arity is zero", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("LPOP")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("errors when too many arguments", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("LPOP", "k", "1", "extra")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("WRONGTYPE when key holds string from SET", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("SET", "k", "v"))
		_, err := redis.DispatchCommand(req("LPOP", "k"))
		if err == nil {
			t.Fatal("want error")
		}
		if !strings.Contains(err.Error(), "WRONGTYPE") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("count larger than length returns whole list as array and removes key", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("RPUSH", "k", "a", "b"))
		got, err := redis.DispatchCommand(req("LPOP", "k", "99"))
		if err != nil {
			t.Fatal(err)
		}
		want := "*2\r\n$1\r\na\r\n$1\r\nb\r\n"
		if string(got) != want {
			t.Fatalf("got %q want %q", got, want)
		}

		got, err = redis.DispatchCommand(req("LPOP", "k"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "$-1\r\n" {
			t.Fatalf("want nil bulk after empty list removed, got %q", got)
		}
	})

	t.Run("errors when count is zero or invalid", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("RPUSH", "k", "x"))
		if _, err := redis.DispatchCommand(req("LPOP", "k", "0")); err == nil {
			t.Fatal("want error for count 0")
		}
		redis.ResetForTesting()
		mustDispatch(t, req("RPUSH", "k", "x"))
		if _, err := redis.DispatchCommand(req("LPOP", "k", "not-int")); err == nil {
			t.Fatal("want error for bad count")
		}
	})
}

func TestBLPOP(t *testing.T) {
	t.Run("matches docs example non-blocking list1 then list2 with timeout zero", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("RPUSH", "list1", "a", "b", "c"))
		got, err := redis.DispatchCommand(req("BLPOP", "list1", "list2", "0"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteArray([]resp.RESP{
			{Type: resp.BulkString, Str: "list1"},
			{Type: resp.BulkString, Str: "a"},
		})
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})

	t.Run("returns first nonempty key scanned left to right", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("RPUSH", "l2", "hello"))
		mustDispatch(t, req("RPUSH", "l3", "bye"))
		got, err := redis.DispatchCommand(req("BLPOP", "l_miss", "l2", "l3", "0"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteArray([]resp.RESP{
			{Type: resp.BulkString, Str: "l2"},
			{Type: resp.BulkString, Str: "hello"},
		})
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})

	t.Run("errors when fewer than key and timeout arguments", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("BLPOP")); err == nil {
			t.Fatal("want error")
		}
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("BLPOP", "onlykey")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("WRONGTYPE when earliest key lists string cache", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("SET", "blocked", "not-a-list"))
		mustDispatch(t, req("RPUSH", "oklist", "x"))
		_, err := redis.DispatchCommand(req("BLPOP", "blocked", "oklist", "0"))
		if err == nil {
			t.Fatal("want WRONGTYPE from first scanned key holding wrong type")
		}
		if !strings.Contains(err.Error(), "WRONGTYPE") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("variadic_lpush_queues_head_like_redis_2_6_then_blpop_returns_leftmost_element", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("LPUSH", "foo", "a", "b", "c"))
		got, err := redis.DispatchCommand(req("BLPOP", "foo", "0"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteArray([]resp.RESP{
			{Type: resp.BulkString, Str: "foo"},
			{Type: resp.BulkString, Str: "c"},
		})
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})

	t.Run("timeoutExpires_returns_null_array_when_no_push", func(t *testing.T) {
		if testing.Short() {
			t.Skip("blocks up to 1s waiting for expiry")
		}
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("BLPOP", "ghost", "1"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "*-1\r\n" {
			t.Fatalf("after timeout Redis returns null array reply; got %q want *-1\\r\\n", got)
		}
	})
}

func TestLPUSH(t *testing.T) {
	t.Run("errors when no element to push", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("LPUSH", "k")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("returns integer length after variadic push", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("LPUSH", "k", "a", "b"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":2\r\n" {
			t.Fatalf("got %q", got)
		}
	})
}
