package redis

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

var listMu sync.Mutex

type blpopWaiter struct {
	keys []string
	ch   chan []byte // capacity 1: RESP array [key][value]
}

var blpopWaiters []*blpopWaiter

func encodeBLPOPReply(key, val string) []byte {
	return resp.WriteArray([]resp.RESP{
		{Type: resp.BulkString, Str: key},
		{Type: resp.BulkString, Str: val},
	})
}

// firstNonEmptyListKey returns first key that has Len>0 when scanned left→right,
// respecting only list keys / missing keys (empty). Caller must hold listMu.
func firstNonEmptyListKey(keys []string) string {
	for _, k := range keys {
		if len(lists[k]) > 0 {
			return k
		}
	}
	return ""
}

func blpopTryPop(keys []string) ([]byte, bool) {
	for _, k := range keys {
		popped := getPoppedElements(k, 1)
		if len(popped) == 1 {
			return encodeBLPOPReply(k, popped[0]), true
		}
	}
	return nil, false
}

// tryServeOneBLPOPWaiter tries to serve one BLPOP waiter by popping one element from the first non-empty list key. It returns true if a waiter was served, false otherwise.
func tryServeOneBLPOPWaiter() bool {
	for i, w := range blpopWaiters {
		k := firstNonEmptyListKey(w.keys) // find the first non-empty list key
		if k == "" {
			continue
		}
		popped := getPoppedElements(k, 1) // pop one element from the list
		if len(popped) != 1 {
			continue
		}
		encoded := encodeBLPOPReply(k, popped[0])
		blpopWaiters = append(blpopWaiters[:i], blpopWaiters[i+1:]...)
		select {
		case w.ch <- encoded:
		default:
			// Buffered ch of size 1 should never block; safeguard no-op drop
		}
		return true
	}
	return false
}

// flushBLPOPAfterPush wakes FIFO waiters until no blocked client can be served with current lists.
func flushBLPOPAfterPush() {
	for tryServeOneBLPOPWaiter() {
	}
}

func removeWaiterIfQueued(w *blpopWaiter) {
	for i, x := range blpopWaiters {
		if x == w {
			blpopWaiters = append(blpopWaiters[:i], blpopWaiters[i+1:]...)
			return
		}
	}
}

// BLPOP implements blocking list pop with global FIFO waiter ordering among clients.
func BLPOP(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'BLPOP'")
	}

	timeoutRaw := args[len(args)-1]
	keys := args[:len(args)-1]
	tsec, err := strconv.ParseFloat(timeoutRaw, 64)
	if err != nil || tsec < 0 {
		return nil, fmt.Errorf("invalid timeout argument for 'BLPOP'")
	}

	listMu.Lock() // lock the listMu to protect the lists map
	for _, k := range keys {
		if _, wrong := cache[k]; wrong {
			listMu.Unlock()
			return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	}

	b, ok := blpopTryPop(keys)
	if ok {
		listMu.Unlock()
		return b, nil
	}

	ch := make(chan []byte, 1)
	w := &blpopWaiter{keys: append([]string(nil), keys...), ch: ch}
	blpopWaiters = append(blpopWaiters, w)
	listMu.Unlock()

	if tsec > 0 {
		timer := time.NewTimer(time.Duration(tsec * float64(time.Second)))
		defer timer.Stop()
		select {
		case reply := <-ch:
			return reply, nil
		case <-timer.C:
			listMu.Lock()
			removeWaiterIfQueued(w)
			listMu.Unlock()
			return []byte("*-1" + resp.CRLF), nil
		}
	}

	reply := <-ch
	return reply, nil
}
