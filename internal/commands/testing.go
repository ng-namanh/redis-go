package commands

func ResetForTesting() {
	mutex.Lock()
	defer mutex.Unlock()
	cache = make(map[string]any)
	lists = make(map[string]list)
	streams = make(map[string]*Stream)
	keyVersions = make(map[string]uint64)
	blpopWaiters = nil
	MasterReplOffset = 0
	Role = "master"
	if aofFile != nil {
		aofFile.Close()
		aofFile = nil
	}
}
