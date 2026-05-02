package redis_test

import (
	"testing"

	"github.com/ng-namanh/redis-go/internal/redis"
)

func TestUNKNOWN(t *testing.T) {
	t.Run("returns error for unknown command name", func(t *testing.T) {
		redis.ResetForTesting()
		if _, err := redis.DispatchCommand(req("NOPE")); err == nil {
			t.Fatal("want error")
		}
	})
}
