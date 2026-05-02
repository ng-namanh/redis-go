package redis

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ng-namanh/redis-go/internal/resp"
)

func resetStores() {
	listMu.Lock()
	defer listMu.Unlock()
	cache = make(map[string]any)
	lists = make(map[string]list)
	streams = make(map[string]*Stream)
	blpopWaiters = nil
}

func req(parts ...string) resp.RESP {
	elems := make([]resp.RESP, len(parts))
	for i, p := range parts {
		elems[i] = resp.RESP{Type: resp.BulkString, Str: p}
	}
	return resp.RESP{Type: resp.Array, Elems: elems}
}

func TestPING(t *testing.T) {
	t.Run("returns simple string PONG", func(t *testing.T) {
		resetStores()
		got, err := DispatchCommand(req("ping"))
		if err != nil {
			t.Fatal(err)
		}
		want := "+PONG\r\n"
		if string(got) != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}

func TestECHO(t *testing.T) {
	t.Run("returns bulk string for single argument", func(t *testing.T) {
		resetStores()
		got, err := DispatchCommand(req("ECHO", "Hello"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteBulkString("Hello")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("errors when no arguments", func(t *testing.T) {
		resetStores()
		if _, err := DispatchCommand(req("ECHO")); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestSET(t *testing.T) {
	t.Run("errors when fewer than key and value", func(t *testing.T) {
		resetStores()
		if _, err := DispatchCommand(req("SET", "onlykey")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("stores value and replies OK", func(t *testing.T) {
		resetStores()
		got, err := DispatchCommand(req("SET", "k", "v"))
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
		resetStores()
		mustDispatch(t, req("SET", "k", "v"))
		got, err := DispatchCommand(req("GET", "k"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteBulkString("v")) {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("returns null bulk when key missing", func(t *testing.T) {
		resetStores()
		got, err := DispatchCommand(req("GET", "missing"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "$-1\r\n" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("errors when stored value is not a string", func(t *testing.T) {
		resetStores()
		cache["n"] = 42
		if _, err := DispatchCommand(req("GET", "n")); err == nil {
			t.Fatal("want error")
		}
	})
}

// TestTYPE specifies behavior from docs/type.md (RESP2 simple string: type name or "none").
// Implement TYPE in package redis and register it in DispatchCommand; these tests call TYPE directly.
func TestTYPE(t *testing.T) {
	t.Run("errors when arity is not exactly one key", func(t *testing.T) {
		resetStores()
		if _, err := TYPE([]string{}); err == nil {
			t.Fatal("want error for zero arguments")
		}
		resetStores()
		if _, err := TYPE([]string{"a", "b"}); err == nil {
			t.Fatal("want error for two arguments")
		}
	})

	t.Run("missing key returns simple string none", func(t *testing.T) {
		resetStores()
		got, err := TYPE([]string{"does-not-exist"})
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteSimpleString("none")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("key from SET is string", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("SET", "k1", "v"))
		got, err := TYPE([]string{"k1"})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteSimpleString("string")) {
			t.Fatalf("got %q, want +string\\r\\n", got)
		}
	})

	t.Run("key from list operations is list", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("RPUSH", "mylist", "x"))
		got, err := TYPE([]string{"mylist"})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteSimpleString("list")) {
			t.Fatalf("got %q, want +list\\r\\n", got)
		}
	})

	t.Run("list type after LPUSH only", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("LPUSH", "other", "y"))
		got, err := TYPE([]string{"other"})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteSimpleString("list")) {
			t.Fatalf("got %q, want +list\\r\\n", got)
		}
	})

	t.Run("stream key returns stream", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("XADD", "s", "0-1", "foo", "bar"))
		got, err := TYPE([]string{"s"})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteSimpleString("stream")) {
			t.Fatalf("got %q, want +stream\\r\\n", got)
		}
	})
}

// TestXADD specifies behavior from docs/xadd.md (explicit ID, bulk string reply; TYPE is stream).
func TestXADD(t *testing.T) {
	t.Run("errors when fewer than key id and one field pair", func(t *testing.T) {
		resetStores()
		if _, err := XADD([]string{"sk"}); err == nil {
			t.Fatal("want error")
		}
		resetStores()
		if _, err := XADD([]string{"sk", "0-1"}); err == nil {
			t.Fatal("want error")
		}
		resetStores()
		if _, err := XADD([]string{"sk", "0-1", "onlyfield"}); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("returns id as bulk string (Codecrafters example 0-1)", func(t *testing.T) {
		resetStores()
		got, err := DispatchCommand(req("XADD", "stream_key", "0-1", "foo", "bar"))
		if err != nil {
			t.Fatal(err)
		}
		want := resp.WriteBulkString("0-1")
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q, want %q (e.g. $3\\r\\n0-1\\r\\n)", got, want)
		}
	})

	t.Run("returns id as bulk string for multi-field entry", func(t *testing.T) {
		resetStores()
		id := "1526919030474-0"
		got, err := DispatchCommand(req("XADD", "weather", id, "temperature", "36", "humidity", "95"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteBulkString(id)) {
			t.Fatalf("got %q, want bulk %q", got, id)
		}
	})

	t.Run("TYPE stream_key is stream after XADD", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("XADD", "stream_key", "0-1", "foo", "bar"))
		got, err := DispatchCommand(req("TYPE", "stream_key"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteSimpleString("stream")) {
			t.Fatalf("got %q, want +stream\\r\\n", got)
		}
	})

	t.Run("creates stream when key did not exist", func(t *testing.T) {
		resetStores()
		_, err := DispatchCommand(req("XADD", "new_stream", "1-0", "k", "v"))
		if err != nil {
			t.Fatal(err)
		}
		got, err := TYPE([]string{"new_stream"})
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, resp.WriteSimpleString("stream")) {
			t.Fatalf("got %q", got)
		}
	})
}

func TestRPUSH(t *testing.T) {
	t.Run("length accumulates across calls", func(t *testing.T) {
		resetStores()
		l1, err := DispatchCommand(req("RPUSH", "fruit", "a"))
		if err != nil {
			t.Fatal(err)
		}
		if string(l1) != ":1\r\n" {
			t.Fatalf("first RPUSH got %s", l1)
		}
		l2, err := DispatchCommand(req("RPUSH", "fruit", "b", "c"))
		if err != nil {
			t.Fatal(err)
		}
		if string(l2) != ":3\r\n" {
			t.Fatalf("got %s", l2)
		}
	})

	t.Run("errors when key but no values", func(t *testing.T) {
		resetStores()
		if _, err := DispatchCommand(req("RPUSH", "k")); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestLRANGE(t *testing.T) {
	t.Run("empty list returns zero-length array", func(t *testing.T) {
		resetStores()
		got, err := DispatchCommand(req("LRANGE", "nix", "0", "-1"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "*0\r\n" {
			t.Fatalf("got %s", got)
		}
	})

	t.Run("errors when fewer than three arguments", func(t *testing.T) {
		resetStores()
		if _, err := DispatchCommand(req("LRANGE", "k", "0")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("full range reflects LPUSH head order after RPUSH tail", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("RPUSH", "k", "tail"))
		mustDispatch(t, req("LPUSH", "k", "mid", "head"))
		got, err := DispatchCommand(req("LRANGE", "k", "0", "-1"))
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
		resetStores()
		got, err := DispatchCommand(req("LLEN", "noSuchKey"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":0\r\n" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("matches docs example after two LPUSH", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("LPUSH", "mylist", "World"))
		mustDispatch(t, req("LPUSH", "mylist", "Hello"))
		got, err := DispatchCommand(req("LLEN", "mylist"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":2\r\n" {
			t.Fatalf("got %q want :2\r\n", got)
		}
	})

	t.Run("totals length after RPUSH with multiple values", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("RPUSH", "fruits", "a"))
		mustDispatch(t, req("RPUSH", "fruits", "b", "c"))
		got, err := DispatchCommand(req("LLEN", "fruits"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":3\r\n" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("reply is RESP integer", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("RPUSH", "solo", "x"))
		got, err := DispatchCommand(req("LLEN", "solo"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":1\r\n" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("exactly one key argument required", func(t *testing.T) {
		resetStores()
		if _, err := DispatchCommand(req("LLEN")); err == nil {
			t.Fatal("want error for arity 0")
		}
		resetStores()
		if _, err := DispatchCommand(req("LLEN", "k", "extra")); err == nil {
			t.Fatal("want error for extra arguments")
		}
	})

	t.Run("WRONGTYPE when key is a string from SET", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("SET", "k", "v"))
		_, err := DispatchCommand(req("LLEN", "k"))
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
		resetStores()
		got, err := DispatchCommand(req("LPOP", "missing"))
		if err != nil {
			t.Fatal(err)
		}
		want := "$-1\r\n"
		if string(got) != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("matches docs example RPUSH LPOP LPOP LRANGE", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("RPUSH", "mylist", "one", "two", "three", "four", "five"))

		got, err := DispatchCommand(req("LPOP", "mylist"))
		if err != nil {
			t.Fatal(err)
		}
		wantOne := resp.WriteBulkString("one")
		if !bytes.Equal(got, wantOne) {
			t.Fatalf("first LPOP got %q want %q", got, wantOne)
		}

		got, err = DispatchCommand(req("LPOP", "mylist", "2"))
		if err != nil {
			t.Fatal(err)
		}
		wantTwoThree := "*2\r\n$3\r\ntwo\r\n$5\r\nthree\r\n"
		if string(got) != wantTwoThree {
			t.Fatalf("LPOP with count:\ngot %q\nwant %q", got, wantTwoThree)
		}

		got, err = DispatchCommand(req("LRANGE", "mylist", "0", "-1"))
		if err != nil {
			t.Fatal(err)
		}
		wantRange := "*2\r\n$4\r\nfour\r\n$4\r\nfive\r\n"
		if string(got) != wantRange {
			t.Fatalf("LRANGE after pops:\ngot %q\nwant %q", got, wantRange)
		}
	})

	t.Run("errors when arity is zero", func(t *testing.T) {
		resetStores()
		if _, err := DispatchCommand(req("LPOP")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("errors when too many arguments", func(t *testing.T) {
		resetStores()
		if _, err := DispatchCommand(req("LPOP", "k", "1", "extra")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("WRONGTYPE when key holds string from SET", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("SET", "k", "v"))
		_, err := DispatchCommand(req("LPOP", "k"))
		if err == nil {
			t.Fatal("want error")
		}
		if !strings.Contains(err.Error(), "WRONGTYPE") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("count larger than length returns whole list as array and removes key", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("RPUSH", "k", "a", "b"))
		got, err := DispatchCommand(req("LPOP", "k", "99"))
		if err != nil {
			t.Fatal(err)
		}
		want := "*2\r\n$1\r\na\r\n$1\r\nb\r\n"
		if string(got) != want {
			t.Fatalf("got %q want %q", got, want)
		}

		got, err = DispatchCommand(req("LPOP", "k"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "$-1\r\n" {
			t.Fatalf("want nil bulk after empty list removed, got %q", got)
		}
	})

	t.Run("errors when count is zero or invalid", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("RPUSH", "k", "x"))
		if _, err := DispatchCommand(req("LPOP", "k", "0")); err == nil {
			t.Fatal("want error for count 0")
		}
		resetStores()
		mustDispatch(t, req("RPUSH", "k", "x"))
		if _, err := DispatchCommand(req("LPOP", "k", "not-int")); err == nil {
			t.Fatal("want error for bad count")
		}
	})
}

func TestBLPOP(t *testing.T) {
	t.Run("matches docs example non-blocking list1 then list2 with timeout zero", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("RPUSH", "list1", "a", "b", "c"))
		got, err := DispatchCommand(req("BLPOP", "list1", "list2", "0"))
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
		resetStores()
		mustDispatch(t, req("RPUSH", "l2", "hello"))
		mustDispatch(t, req("RPUSH", "l3", "bye"))
		got, err := DispatchCommand(req("BLPOP", "l_miss", "l2", "l3", "0"))
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
		resetStores()
		if _, err := DispatchCommand(req("BLPOP")); err == nil {
			t.Fatal("want error")
		}
		resetStores()
		if _, err := DispatchCommand(req("BLPOP", "onlykey")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("WRONGTYPE when earliest key lists string cache", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("SET", "blocked", "not-a-list"))
		mustDispatch(t, req("RPUSH", "oklist", "x"))
		_, err := DispatchCommand(req("BLPOP", "blocked", "oklist", "0"))
		if err == nil {
			t.Fatal("want WRONGTYPE from first scanned key holding wrong type")
		}
		if !strings.Contains(err.Error(), "WRONGTYPE") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("variadic_lpush_queues_head_like_redis_2_6_then_blpop_returns_leftmost_element", func(t *testing.T) {
		resetStores()
		mustDispatch(t, req("LPUSH", "foo", "a", "b", "c"))
		got, err := DispatchCommand(req("BLPOP", "foo", "0"))
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
		resetStores()
		got, err := DispatchCommand(req("BLPOP", "ghost", "1"))
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
		resetStores()
		if _, err := DispatchCommand(req("LPUSH", "k")); err == nil {
			t.Fatal("want error")
		}
	})

	t.Run("returns integer length after variadic push", func(t *testing.T) {
		resetStores()
		got, err := DispatchCommand(req("LPUSH", "k", "a", "b"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != ":2\r\n" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestUNKNOWN(t *testing.T) {
	t.Run("returns error for unknown command name", func(t *testing.T) {
		resetStores()
		if _, err := DispatchCommand(req("NOPE")); err == nil {
			t.Fatal("want error")
		}
	})
}

func TestLrangeSlice(t *testing.T) {
	t.Run("negative stop includes full length", func(t *testing.T) {
		l := list{"a", "b", "c", "d"}
		sub := lrangeSlice(l, 0, -1)
		if len(sub) != 4 || sub[0] != "a" || sub[3] != "d" {
			t.Fatalf("got %v", sub)
		}
	})

	t.Run("mixed positive start and negative stop", func(t *testing.T) {
		l := list{"a", "b", "c", "d"}
		sub := lrangeSlice(l, 1, -2)
		if len(sub) != 2 || sub[0] != "b" || sub[1] != "c" {
			t.Fatalf("got %v", sub)
		}
	})

	t.Run("start past end returns nil", func(t *testing.T) {
		l := list{"a", "b", "c", "d"}
		if sub := lrangeSlice(l, 10, 20); sub != nil {
			t.Fatalf("got %v", sub)
		}
	})

	t.Run("inverted range after normalization returns nil", func(t *testing.T) {
		l := list{"a", "b", "c", "d"}
		if sub := lrangeSlice(l, 3, 1); sub != nil {
			t.Fatalf("got %v", sub)
		}
	})

	t.Run("nil list returns nil", func(t *testing.T) {
		if sub := lrangeSlice(nil, 0, -1); sub != nil {
			t.Fatalf("got %v", sub)
		}
	})
}

func mustDispatch(t *testing.T, v resp.RESP) []byte {
	t.Helper()
	b, err := DispatchCommand(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
