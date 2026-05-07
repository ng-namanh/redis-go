package commands

func ResetForTesting() {
	mutex.Lock()
	defer mutex.Unlock()
	cache = make(map[string]any)
	lists = make(map[string]list)
	streams = make(map[string]*Stream)
	blpopWaiters = nil
}
