package commands

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

type blpopWaiter struct {
	keys []string
	ch   chan []byte
}

var blpopWaiters []*blpopWaiter

func encodeBLPOPReply(key, val string) []byte {
	return resp.WriteArray([]resp.RESP{
		{Type: resp.BulkString, Str: key},
		{Type: resp.BulkString, Str: val},
	})
}

// firstNonEmptyListKey returns the first key that has Len>0 scanned left→right.
// Caller must hold listMu.
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
			Touch(k)
			Propagate("LPOP", []string{k})
			return encodeBLPOPReply(k, popped[0]), true
		}
	}
	return nil, false
}

// tryServeOneBLPOPWaiter tries to serve one BLPOP waiter.
// Returns true if a waiter was served.
func tryServeOneBLPOPWaiter() bool {
	for i, w := range blpopWaiters {
		k := firstNonEmptyListKey(w.keys)
		if k == "" {
			continue
		}
		popped := getPoppedElements(k, 1)
		if len(popped) != 1 {
			continue
		}
		Touch(k)
		Propagate("LPOP", []string{k})
		encoded := encodeBLPOPReply(k, popped[0])
		blpopWaiters = append(blpopWaiters[:i], blpopWaiters[i+1:]...)
		select {
		case w.ch <- encoded:
		default:
		}
		return true
	}
	return false
}

// flushBLPOPAfterPush wakes FIFO waiters until no blocked client can be served.
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

	mutex.Lock()
	for _, k := range keys {
		if _, wrong := cache[k]; wrong {
			mutex.Unlock()
			return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	}

	b, ok := blpopTryPop(keys)
	if ok {
		mutex.Unlock()
		return b, nil
	}

	ch := make(chan []byte, 1)
	w := &blpopWaiter{keys: append([]string(nil), keys...), ch: ch}
	blpopWaiters = append(blpopWaiters, w)
	mutex.Unlock()

	if tsec > 0 {
		timer := time.NewTimer(time.Duration(tsec * float64(time.Second)))
		defer timer.Stop()
		select {
		case reply := <-ch:
			return reply, nil
		case <-timer.C:
			mutex.Lock()
			removeWaiterIfQueued(w)
			mutex.Unlock()
			return []byte("*-1" + resp.CRLF), nil
		}
	}

	reply := <-ch
	return reply, nil
}
