package redis_test

import (
	"testing"

	"github.com/ng-namanh/redis-go/internal/redis"
)

func TestZSET(t *testing.T) {
	t.Run("ZADD and ZCARD", func(t *testing.T) {
		redis.ResetForTesting()
		got := mustDispatch(t, req("ZADD", "myzset", "1", "one"))
		if string(got) != ":1\r\n" {
			t.Fatalf("ZADD got %s", got)
		}
		got = mustDispatch(t, req("ZADD", "myzset", "2", "two", "3", "three"))
		if string(got) != ":2\r\n" {
			t.Fatalf("ZADD got %s", got)
		}
		got = mustDispatch(t, req("ZCARD", "myzset"))
		if string(got) != ":3\r\n" {
			t.Fatalf("ZCARD got %s", got)
		}
	})

	t.Run("ZSCORE", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("ZADD", "myzset", "1.5", "one"))
		got := mustDispatch(t, req("ZSCORE", "myzset", "one"))
		if string(got) != "$3\r\n1.5\r\n" {
			t.Fatalf("ZSCORE got %q", got)
		}
		got = mustDispatch(t, req("ZSCORE", "myzset", "nonexistent"))
		if string(got) != "$-1\r\n" {
			t.Fatalf("ZSCORE nonexistent got %q", got)
		}
	})

	t.Run("ZRANK", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("ZADD", "myzset", "10", "c", "5", "a", "7", "b"))
		// Ordered: a(5), b(7), c(10)
		got := mustDispatch(t, req("ZRANK", "myzset", "a"))
		if string(got) != ":0\r\n" {
			t.Fatalf("ZRANK a got %s", got)
		}
		got = mustDispatch(t, req("ZRANK", "myzset", "b"))
		if string(got) != ":1\r\n" {
			t.Fatalf("ZRANK b got %s", got)
		}
		got = mustDispatch(t, req("ZRANK", "myzset", "c"))
		if string(got) != ":2\r\n" {
			t.Fatalf("ZRANK c got %s", got)
		}
		got = mustDispatch(t, req("ZRANK", "myzset", "nonexistent"))
		if string(got) != "$-1\r\n" {
			t.Fatalf("ZRANK nonexistent got %q", got)
		}
	})

	t.Run("ZRANGE", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("ZADD", "myzset", "1", "one", "2", "two", "3", "three"))

		// ZRANGE 0 -1
		got := mustDispatch(t, req("ZRANGE", "myzset", "0", "-1"))
		want := "*3\r\n$3\r\none\r\n$3\r\ntwo\r\n$5\r\nthree\r\n"
		if string(got) != want {
			t.Fatalf("ZRANGE 0 -1 got %q", got)
		}

		// ZRANGE 0 -1 WITHSCORES
		got = mustDispatch(t, req("ZRANGE", "myzset", "0", "-1", "WITHSCORES"))
		want = "*6\r\n$3\r\none\r\n$1\r\n1\r\n$3\r\ntwo\r\n$1\r\n2\r\n$5\r\nthree\r\n$1\r\n3\r\n"
		if string(got) != want {
			t.Fatalf("ZRANGE 0 -1 WITHSCORES got %q", got)
		}

		// ZRANGE 1 1
		got = mustDispatch(t, req("ZRANGE", "myzset", "1", "1"))
		want = "*1\r\n$3\r\ntwo\r\n"
		if string(got) != want {
			t.Fatalf("ZRANGE 1 1 got %q", got)
		}
	})

	t.Run("ZREM", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("ZADD", "myzset", "1", "one", "2", "two", "3", "three"))
		got := mustDispatch(t, req("ZREM", "myzset", "two", "nonexistent"))
		if string(got) != ":1\r\n" {
			t.Fatalf("ZREM got %s", got)
		}
		got = mustDispatch(t, req("ZCARD", "myzset"))
		if string(got) != ":2\r\n" {
			t.Fatalf("ZCARD after ZREM got %s", got)
		}
	})

	t.Run("Lexicographical order for same scores", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("ZADD", "myzset", "10", "banana", "10", "apple", "10", "cherry"))
		// Should be: apple, banana, cherry
		got := mustDispatch(t, req("ZRANGE", "myzset", "0", "-1"))
		want := "*3\r\n$5\r\napple\r\n$6\r\nbanana\r\n$6\r\ncherry\r\n"
		if string(got) != want {
			t.Fatalf("Lex order got %q", got)
		}
	})
}
