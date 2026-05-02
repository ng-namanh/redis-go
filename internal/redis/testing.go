package redis

// ResetForTesting clears in-memory string, list, and stream state and blocking waiters.
// It is intended for tests (package redis_test under internal/redis/test).
func ResetForTesting() {
	listMu.Lock()
	defer listMu.Unlock()
	cache = make(map[string]any)
	lists = make(map[string]list)
	streams = make(map[string]*Stream)
	blpopWaiters = nil
}
