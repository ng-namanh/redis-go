package commands

import (
	"fmt"
	"strconv"

	"github.com/ng-namanh/redis-go/internal/resp"
)

type SortedSet struct {
	kv  map[string]float64
	zsl *SkipList
}

func NewSortedSet() *SortedSet {
	return &SortedSet{
		kv:  make(map[string]float64),
		zsl: NewSkipList(),
	}
}

func (zs *SortedSet) add(member string, score float64) int {
	oldScore, exists := zs.kv[member]
	if exists {
		if oldScore == score {
			return 0
		}
		// Update: delete and re-insert
		zs.zsl.delete(oldScore, member)
		zs.kv[member] = score
		zs.zsl.insert(score, member)
		return 0
	}

	zs.kv[member] = score
	zs.zsl.insert(score, member)
	return 1
}

func (zs *SortedSet) rem(member string) int {
	score, exists := zs.kv[member]
	if !exists {
		return 0
	}

	delete(zs.kv, member)
	zs.zsl.delete(score, member)
	return 1
}

func (zs *SortedSet) rank(member string) (int64, bool) {
	score, exists := zs.kv[member]
	if !exists {
		return 0, false
	}
	r := zs.zsl.getRank(score, member)
	if r == 0 {
		return 0, false
	}
	return int64(r) - 1, true
}

func (zs *SortedSet) rangeByIndex(start, stop int) []*SkipNode {
	ln := int64(zs.zsl.length)
	if ln == 0 {
		return nil
	}

	s := int64(start)
	e := int64(stop)

	if s < 0 {
		s += ln
	}
	if s < 0 {
		s = 0
	}
	if e < 0 {
		e += ln
	}
	if e >= ln {
		e = ln - 1
	}

	if s > e || s >= ln {
		return nil
	}

	nodes := make([]*SkipNode, 0, e-s+1)
	// Get first node in range
	x := zs.zsl.getElementByRank(uint64(s + 1))
	for i := s; i <= e && x != nil; i++ {
		nodes = append(nodes, x)
		x = x.level[0].forward
	}
	return nodes
}

// ZADD key score member [score member ...]
func ZADD(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return zaddUnlocked(args)
}

func zaddUnlocked(args []string) ([]byte, error) {
	if len(args) < 3 || (len(args)-1)%2 != 0 {
		return nil, fmt.Errorf("wrong number of arguments for 'ZADD'")
	}

	key := args[0]
	zs, ok := sortedSets[key]
	if !ok {
		zs = NewSortedSet()
		sortedSets[key] = zs
	}

	added := 0
	for i := 1; i < len(args); i += 2 {
		score, err := strconv.ParseFloat(args[i], 64)
		if err != nil {
			return nil, fmt.Errorf("value is not a valid float")
		}
		member := args[i+1]
		added += zs.add(member, score)
	}

	Touch(key)
	Propagate("ZADD", args)
	return resp.WriteInteger(int64(added)), nil
}

// ZRANK key member
func ZRANK(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return zrankUnlocked(args)
}

func zrankUnlocked(args []string) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'ZRANK'")
	}

	key := args[0]
	member := args[1]

	zs, ok := sortedSets[key]
	if !ok {
		return []byte("$-1" + resp.CRLF), nil
	}

	rank, exists := zs.rank(member)
	if !exists {
		return []byte("$-1" + resp.CRLF), nil
	}

	return resp.WriteInteger(rank), nil
}

// ZRANGE key start stop [WITHSCORES]
func ZRANGE(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return zrangeUnlocked(args)
}

func zrangeUnlocked(args []string) ([]byte, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("wrong number of arguments for 'ZRANGE'")
	}

	key := args[0]
	start, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, fmt.Errorf("value is not an integer or out of range")
	}
	stop, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, fmt.Errorf("value is not an integer or out of range")
	}

	withScores := false
	if len(args) > 3 {
		if args[3] == "WITHSCORES" {
			withScores = true
		} else {
			return nil, fmt.Errorf("syntax error")
		}
	}

	zs, ok := sortedSets[key]
	if !ok {
		return resp.WriteArray([]resp.RESP{}), nil
	}

	nodes := zs.rangeByIndex(start, stop)
	elems := make([]resp.RESP, 0)
	for _, n := range nodes {
		elems = append(elems, resp.RESP{Type: resp.BulkString, Str: n.member})
		if withScores {
			scoreStr := strconv.FormatFloat(n.score, 'f', -1, 64)
			elems = append(elems, resp.RESP{Type: resp.BulkString, Str: scoreStr})
		}
	}

	return resp.WriteArray(elems), nil
}

// ZCARD key
func ZCARD(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return zcardUnlocked(args)
}

func zcardUnlocked(args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("wrong number of arguments for 'ZCARD'")
	}

	key := args[0]
	zs, ok := sortedSets[key]
	if !ok {
		return resp.WriteInteger(0), nil
	}

	return resp.WriteInteger(int64(zs.zsl.length)), nil
}

// ZSCORE key member
func ZSCORE(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return zscoreUnlocked(args)
}

func zscoreUnlocked(args []string) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'ZSCORE'")
	}

	key := args[0]
	member := args[1]

	zs, ok := sortedSets[key]
	if !ok {
		return []byte("$-1" + resp.CRLF), nil
	}

	score, exists := zs.kv[member]
	if !exists {
		return []byte("$-1" + resp.CRLF), nil
	}

	return resp.WriteBulkString(strconv.FormatFloat(score, 'f', -1, 64)), nil
}

// ZREM key member [member ...]
func ZREM(args []string) ([]byte, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return zremUnlocked(args)
}

func zremUnlocked(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'ZREM'")
	}

	key := args[0]
	zs, ok := sortedSets[key]
	if !ok {
		return resp.WriteInteger(0), nil
	}

	removed := 0
	for i := 1; i < len(args); i++ {
		removed += zs.rem(args[i])
	}

	if zs.zsl.length == 0 {
		delete(sortedSets, key)
	}

	Touch(key)
	Propagate("ZREM", args)
	return resp.WriteInteger(int64(removed)), nil
}
