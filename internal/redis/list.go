package redis

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ng-namanh/redis-go/internal/resp"
)

type list []string

var lists map[string]list = make(map[string]list)

// lrangeSlice returns Redis LRANGE-inclusive elements for start/stop (negative indices OK).
func lrangeSlice(l list, start, stop int) list {
	ln := len(l)
	if ln == 0 {
		return nil
	}

	startIdx := start

	if startIdx >= ln {
		return nil
	}

	if startIdx < 0 {
		startIdx += ln

		if startIdx < 0 {
			startIdx = 0
		}
	}

	stopIdx := stop
	if stopIdx < 0 {
		stopIdx += ln
		if stopIdx < 0 {
			stopIdx = 0
		}
	}

	if stopIdx >= ln {
		stopIdx = ln - 1
	}

	if startIdx > stopIdx {
		return nil
	}

	return l[startIdx : stopIdx+1]
}

func listsLen(key string) int {
	return len(lists[key])
}

func listsRPush(key string, elems []string) int {
	if _, ok := lists[key]; !ok {
		lists[key] = make(list, 0)
	}
	lists[key] = append(lists[key], elems...)
	return len(lists[key])
}

func listsLPush(key string, vals []string) int {
	old := lists[key]
	n := len(vals) + len(old) // new length of the list
	prefix := make(list, n)   // prefix is the new list
	at := 0                   // at is the index of the first element to push
	for i := len(vals) - 1; i >= 0; i-- {
		prefix[at] = vals[i] // push the values to the prefix
		at++
	}
	copy(prefix[at:], old)
	lists[key] = prefix
	return len(lists[key])
}

func getPoppedElements(key string, n int) []string {
	lst := lists[key]
	if len(lst) == 0 || n <= 0 {
		return nil
	}

	at := min(n, len(lst))
	popped := append(list(nil), lst[:at]...)
	lists[key] = append(list(nil), lst[at:]...)

	if len(lists[key]) == 0 {
		delete(lists, key)
	}
	return popped
}

func RPUSH(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'RPUSH'")
	}

	listName := args[0]
	values := append([]string(nil), args[1:]...)

	listMu.Lock()
	defer listMu.Unlock()

	n := listsRPush(listName, values)
	flushBLPOPAfterPush() // wake up the blocked clients
	return resp.WriteInteger(int64(n)), nil
}

func LRANGE(args []string) ([]byte, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'LRANGE'")
	}

	listName := args[0]
	start, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, fmt.Errorf("invalid argument for 'LRANGE'")
	}
	end, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, fmt.Errorf("invalid argument for 'LRANGE'")
	}

	listMu.Lock()
	defer listMu.Unlock()

	sub := lrangeSlice(lists[listName], start, end)
	elements := make([]resp.RESP, 0, len(sub))
	for _, s := range sub {
		elements = append(elements, resp.RESP{Type: resp.BulkString, Str: s})
	}
	return resp.WriteArray(elements), nil
}

// Insert all the specified values at the head of the list stored at key.
// If key does not exist, it is created as empty list before performing the push operations.
// When key holds a value that is not a list, an error is returned.
func LPUSH(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'LPUSH'")
	}

	listName := args[0]
	vals := append([]string(nil), args[1:]...) // values to push to the list

	listMu.Lock()
	defer listMu.Unlock()

	n := listsLPush(listName, vals)
	flushBLPOPAfterPush()
	return resp.WriteInteger(int64(n)), nil
}

func LLEN(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'LLEN'")
	}

	listName := args[0]
	listMu.Lock()
	defer listMu.Unlock()

	if _, holdsKey := cache[listName]; holdsKey {
		return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}
	return resp.WriteInteger(int64(listsLen(listName))), nil
}

func LPOP(args []string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'LPOP'")
	}
	if len(args) > 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'LPOP'")
	}

	key := args[0]
	if _, inCache := cache[key]; inCache {
		return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
	}

	listMu.Lock()
	defer listMu.Unlock()

	if listsLen(key) == 0 {
		return []byte("$-1" + resp.CRLF), nil
	}

	if len(args) == 1 {
		popped := getPoppedElements(key, 1)
		return resp.WriteBulkString(popped[0]), nil
	}

	count, err := strconv.Atoi(args[1])
	if err != nil || count <= 0 {
		return nil, fmt.Errorf("invalid argument for 'LPOP'")
	}

	popped := getPoppedElements(key, count)
	out := make([]resp.RESP, 0, len(popped))
	for _, s := range popped {
		out = append(out, resp.RESP{Type: resp.BulkString, Str: s})
	}
	return resp.WriteArray(out), nil
}

func deleteKeyAfterDuration(key string, duration int64) {
	time.Sleep(time.Duration(duration) * time.Millisecond)
	delete(cache, key)
}
