package redis_test

import (
	"bufio"
	"bytes"
	"testing"
	"time"

	"github.com/ng-namanh/redis-go/internal/redis"
	"github.com/ng-namanh/redis-go/internal/resp"
)

func TestXREAD(t *testing.T) {
	t.Run("reads entries with IDs greater than the provided ID", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "s", "0-1", "a", "1"))
		mustDispatch(t, req("XADD", "s", "0-2", "b", "2"))

		got, err := redis.DispatchCommand(req("XREAD", "STREAMS", "s", "0-0"))
		if err != nil {
			t.Fatal(err)
		}

		want := resp.WriteArray([]resp.RESP{
			{
				Type: resp.Array,
				Elems: []resp.RESP{
					{Type: resp.BulkString, Str: "s"},
					{Type: resp.Array, Elems: []resp.RESP{
						{
							Type: resp.Array,
							Elems: []resp.RESP{
								{Type: resp.BulkString, Str: "0-1"},
								{Type: resp.Array, Elems: []resp.RESP{
									{Type: resp.BulkString, Str: "a"},
									{Type: resp.BulkString, Str: "1"},
								}},
							},
						},
						{
							Type: resp.Array,
							Elems: []resp.RESP{
								{Type: resp.BulkString, Str: "0-2"},
								{Type: resp.Array, Elems: []resp.RESP{
									{Type: resp.BulkString, Str: "b"},
									{Type: resp.BulkString, Str: "2"},
								}},
							},
						},
					}},
				},
			},
		})
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})

	t.Run("COUNT limits number of entries per stream", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "s", "0-1", "a", "1"))
		mustDispatch(t, req("XADD", "s", "0-2", "b", "2"))
		mustDispatch(t, req("XADD", "s", "0-3", "c", "3"))

		got, err := redis.DispatchCommand(req("XREAD", "COUNT", "2", "STREAMS", "s", "0-0"))
		if err != nil {
			t.Fatal(err)
		}

		// Only first two entries.
		want := resp.WriteArray([]resp.RESP{
			{
				Type: resp.Array,
				Elems: []resp.RESP{
					{Type: resp.BulkString, Str: "s"},
					{Type: resp.Array, Elems: []resp.RESP{
						{
							Type: resp.Array,
							Elems: []resp.RESP{
								{Type: resp.BulkString, Str: "0-1"},
								{Type: resp.Array, Elems: []resp.RESP{
									{Type: resp.BulkString, Str: "a"},
									{Type: resp.BulkString, Str: "1"},
								}},
							},
						},
						{
							Type: resp.Array,
							Elems: []resp.RESP{
								{Type: resp.BulkString, Str: "0-2"},
								{Type: resp.Array, Elems: []resp.RESP{
									{Type: resp.BulkString, Str: "b"},
									{Type: resp.BulkString, Str: "2"},
								}},
							},
						},
					}},
				},
			},
		})
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})

	t.Run("supports milliseconds-only IDs (sequence treated as 0)", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "s", "5-0", "k", "v"))
		got, err := redis.DispatchCommand(req("XREAD", "STREAMS", "s", "5"))
		if err != nil {
			t.Fatal(err)
		}
		// Last seen is 5-0, so there are no greater IDs.
		if string(got) != "*0\r\n" {
			t.Fatalf("got %q want *0\\r\\n", got)
		}
	})

	t.Run("special $ returns only entries added after the call (non-blocking yields empty)", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "s", "0-1", "a", "1"))
		got, err := redis.DispatchCommand(req("XREAD", "STREAMS", "s", "$"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "*0\r\n" {
			t.Fatalf("got %q want *0\\r\\n", got)
		}
	})

	t.Run("BLOCK returns nil reply on timeout when no stream can be served", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "s", "0-1", "a", "1"))
		got, err := redis.DispatchCommand(req("XREAD", "BLOCK", "30", "STREAMS", "s", "$"))
		if err != nil {
			t.Fatal(err)
		}
		v, err := resp.ReadValue(bufio.NewReader(bytes.NewReader(got)))
		if err != nil {
			t.Fatal(err)
		}
		if v.Type != resp.BulkString || !v.Null {
			t.Fatalf("want null bulk string (nil reply), got %+v", v)
		}
	})

	t.Run("BLOCK with $ unblocks when a new entry is added", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "s", "0-1", "a", "1"))

		done := make(chan []byte, 1)
		go func() {
			b, _ := redis.DispatchCommand(req("XREAD", "BLOCK", "200", "STREAMS", "s", "$"))
			done <- b
		}()

		time.Sleep(30 * time.Millisecond)
		mustDispatch(t, req("XADD", "s", "0-2", "b", "2"))

		got := <-done
		// Should return the new entry 0-2.
		want := resp.WriteArray([]resp.RESP{
			{
				Type: resp.Array,
				Elems: []resp.RESP{
					{Type: resp.BulkString, Str: "s"},
					{Type: resp.Array, Elems: []resp.RESP{
						{
							Type: resp.Array,
							Elems: []resp.RESP{
								{Type: resp.BulkString, Str: "0-2"},
								{Type: resp.Array, Elems: []resp.RESP{
									{Type: resp.BulkString, Str: "b"},
									{Type: resp.BulkString, Str: "2"},
								}},
							},
						},
					}},
				},
			},
		})
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})

	t.Run("special + returns the last available entry for each stream", func(t *testing.T) {
		redis.ResetForTesting()
		mustDispatch(t, req("XADD", "s", "0-1", "a", "1"))
		mustDispatch(t, req("XADD", "s", "0-2", "b", "2"))

		got, err := redis.DispatchCommand(req("XREAD", "STREAMS", "s", "+"))
		if err != nil {
			t.Fatal(err)
		}

		want := resp.WriteArray([]resp.RESP{
			{
				Type: resp.Array,
				Elems: []resp.RESP{
					{Type: resp.BulkString, Str: "s"},
					{Type: resp.Array, Elems: []resp.RESP{
						{
							Type: resp.Array,
							Elems: []resp.RESP{
								{Type: resp.BulkString, Str: "0-2"},
								{Type: resp.Array, Elems: []resp.RESP{
									{Type: resp.BulkString, Str: "b"},
									{Type: resp.BulkString, Str: "2"},
								}},
							},
						},
					}},
				},
			},
		})
		if !bytes.Equal(got, want) {
			t.Fatalf("got %q\nwant %q", got, want)
		}
	})
}
