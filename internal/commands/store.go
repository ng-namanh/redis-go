package commands

import (
	"math/rand"
	"net"
	"sync"

	"github.com/ng-namanh/redis-go/internal/resp"
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

var replicas []net.Conn
var replicasMutex sync.Mutex

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

func AddReplica(conn net.Conn) {
	replicasMutex.Lock()
	defer replicasMutex.Unlock()
	replicas = append(replicas, conn)
}

func Propagate(cmd string, args []string) {
	if Role != "master" {
		return
	}

	// Only propagate if we have replicas
	replicasMutex.Lock()
	if len(replicas) == 0 {
		replicasMutex.Unlock()
		return
	}
	defer replicasMutex.Unlock()

	// Encode command as RESP array of bulk strings
	elems := make([]resp.RESP, len(args)+1)
	elems[0] = resp.RESP{Type: resp.BulkString, Str: cmd}
	for i, arg := range args {
		elems[i+1] = resp.RESP{Type: resp.BulkString, Str: arg}
	}
	wire := resp.WriteArray(elems)

	for _, r := range replicas {
		_, _ = r.Write(wire)
	}
}
