package redis_test

import (
	"testing"

	"github.com/ng-namanh/redis-go/internal/redis"
)

func TestPING(t *testing.T) {
	t.Run("returns simple string PONG", func(t *testing.T) {
		redis.ResetForTesting()
		got, err := redis.DispatchCommand(req("ping"))
		if err != nil {
			t.Fatal(err)
		}
		want := "+PONG\r\n"
		if string(got) != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})
}
