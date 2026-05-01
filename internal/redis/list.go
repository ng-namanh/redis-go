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
	n := len(vals) + len(old)
	prefix := make(list, n)
	at := 0
	for i := len(vals) - 1; i >= 0; i-- {
		prefix[at] = vals[i]
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
	capN := min(n, len(lst))
	popped := append(list(nil), lst[:capN]...)
	lists[key] = append(list(nil), lst[capN:]...)
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

	n := listsRPush(listName, values)
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

	sub := lrangeSlice(lists[listName], start, end)
	elements := make([]resp.RESP, 0, len(sub))
	for _, s := range sub {
		elements = append(elements, resp.RESP{Type: resp.BulkString, Str: s})
	}
	return resp.WriteArray(elements), nil
}

func LPUSH(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'LPUSH'")
	}

	listName := args[0]
	vals := append([]string(nil), args[1:]...)

	n := listsLPush(listName, vals)
	return resp.WriteInteger(int64(n)), nil
}

func LLEN(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'LLEN'")
	}

	listName := args[0]
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

// BLPOP implements sync semantics matching tests: first non-empty list wins;
// if timeout > 0 and nothing to pop initially, sleeps then retries once before nil array reply.
func BLPOP(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'BLPOP'")
	}

	timeoutRaw := args[len(args)-1]                 // Last argument is the timeout
	keys := args[:len(args)-1]                      // Remaining arguments except the timeout are the keys
	tsec, err := strconv.ParseFloat(timeoutRaw, 64) // Parse the timeout as a float

	if err != nil || tsec < 0 {
		return nil, fmt.Errorf("invalid argument for 'BLPOP'")
	}

	// check if the provided keys are in the lists map
	for _, k := range keys {
		if _, in := lists[k]; !in {
			return nil, fmt.Errorf("WRONGTYPE Operation against a key holding the wrong kind of value")
		}
	}

	tryFirst := func() ([]byte, error) {
		for _, k := range keys {
			popped := getPoppedElements(k, 1)
			if len(popped) == 1 {
				return resp.WriteArray([]resp.RESP{
					{Type: resp.BulkString, Str: k},
					{Type: resp.BulkString, Str: popped[0]},
				}), nil
			}
		}
		return nil, nil
	}

	b, err := tryFirst()
	if err != nil {
		return nil, err
	}
	if b != nil {
		return b, nil
	}

	if tsec > 0 {
		time.Sleep(time.Duration(tsec * float64(time.Second)))
		b, err = tryFirst()
		if err != nil {
			return nil, err
		}
		if b != nil {
			return b, nil
		}
		return []byte("*-1" + resp.CRLF), nil
	}

	for {
		b, err := tryFirst()
		if err != nil {
			return nil, err
		}
		if b != nil {
			return b, nil
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func deleteKeyAfterDuration(key string, duration int64) {
	time.Sleep(time.Duration(duration) * time.Millisecond)
	delete(cache, key)
}
