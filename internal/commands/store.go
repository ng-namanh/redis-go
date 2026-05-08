package commands

import "sync"

var mutex sync.Mutex
var cache = map[string]any{}
var lists map[string]list = make(map[string]list)
var streams = make(map[string]*Stream)
var Role = "master"
var keyVersions = make(map[string]uint64)

func Lock() {
	mutex.Lock()
}

func Unlock() {
	mutex.Unlock()
}

// Touch increments the version of a key. Caller must hold mutex.
func Touch(key string) {
	keyVersions[key]++
}

// GetVersion returns the current version of a key. Caller must hold mutex.
func GetVersion(key string) uint64 {
	return keyVersions[key]
}
