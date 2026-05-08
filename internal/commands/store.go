package commands

import (
	"math/rand"
	"sync"
)

var mutex sync.Mutex
var cache = map[string]any{}
var lists map[string]list = make(map[string]list)
var streams = make(map[string]*Stream)
var Role = "master"
var MasterReplid string
var MasterReplOffset int64 = 0
var MasterHost string
var MasterPort string
var ServerPort string
var keyVersions = make(map[string]uint64)

func init() {
	MasterReplid = generateRandomID(40)
}

func generateRandomID(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func Lock() {
	mutex.Lock()
}

func Unlock() {
	mutex.Unlock()
}

// increments the version of a key. Caller must hold mutex.
func Touch(key string) {
	keyVersions[key]++
}

// returns the current version of a key. Caller must hold mutex.
func GetVersion(key string) uint64 {
	return keyVersions[key]
}
