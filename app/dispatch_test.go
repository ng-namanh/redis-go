package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ng-namanh/redis-go/internal/resp"
)

func resetStores() {
	cache = make(map[string]any)
	lists = make(map[string]list)
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
