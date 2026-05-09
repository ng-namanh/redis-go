package redis_test

import (
	"bytes"
	"testing"

	"github.com/ng-namanh/redis-go/internal/commands"
	"github.com/ng-namanh/redis-go/internal/resp"
)

func TestConfigGet(t *testing.T) {
	commands.Dir = "/tmp/redis"
	commands.Dbfilename = "dump.rdb"

	t.Run("CONFIG GET dir", func(t *testing.T) {
		res, err := commands.CONFIG([]string{"GET", "dir"})
		if err != nil {
			t.Fatalf("CONFIG GET dir failed: %v", err)
		}
		
		want := resp.WriteArray([]resp.RESP{
			{Type: resp.BulkString, Str: "dir"},
			{Type: resp.BulkString, Str: "/tmp/redis"},
		})
		if !bytes.Equal(res, want) {
			t.Fatalf("got %q, want %q", res, want)
		}
	})

	t.Run("CONFIG GET dbfilename", func(t *testing.T) {
		res, err := commands.CONFIG([]string{"GET", "dbfilename"})
		if err != nil {
			t.Fatalf("CONFIG GET dbfilename failed: %v", err)
		}
		
		want := resp.WriteArray([]resp.RESP{
			{Type: resp.BulkString, Str: "dbfilename"},
			{Type: resp.BulkString, Str: "dump.rdb"},
		})
		if !bytes.Equal(res, want) {
			t.Fatalf("got %q, want %q", res, want)
		}
	})
}
