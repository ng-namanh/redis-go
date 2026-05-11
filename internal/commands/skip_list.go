package commands

import "math/rand"

const (
	zskipListMaxLevel = 32
	zskipListP        = 0.25
)

type SkipLevel struct {
	forward *SkipNode
	span    uint64
}

type SkipNode struct {
	member   string
	score    float64
	backward *SkipNode
	level    []SkipLevel
}

type SkipList struct {
	header, tail *SkipNode
	length       uint64
	level        int
}

func NewSkipNode(level int, score float64, member string) *SkipNode {
	return &SkipNode{
		member: member,
		score:  score,
		level:  make([]SkipLevel, level),
	}
}

func NewSkipList() *SkipList {
	zsl := &SkipList{
		level: 1,
	}
	zsl.header = NewSkipNode(zskipListMaxLevel, 0, "")
	return zsl
}

func (zsl *SkipList) randomLevel() int {
	level := 1
	for float64(rand.Int31()&0xFFFF) < (zskipListP * 0xFFFF) {
		level++
	}
	if level < zskipListMaxLevel {
		return level
	}
	return zskipListMaxLevel
}

func (zsl *SkipList) insert(score float64, member string) *SkipNode {
	update := make([]*SkipNode, zskipListMaxLevel)
	rank := make([]uint64, zskipListMaxLevel)
	x := zsl.header

	for i := zsl.level - 1; i >= 0; i-- {
		if i == zsl.level-1 {
			rank[i] = 0
		} else {
			rank[i] = rank[i+1]
		}
		for x.level[i].forward != nil &&
			(x.level[i].forward.score < score ||
				(x.level[i].forward.score == score && x.level[i].forward.member < member)) {
			rank[i] += x.level[i].span
			x = x.level[i].forward
		}
		update[i] = x
	}

	level := zsl.randomLevel()
	if level > zsl.level {
		for i := zsl.level; i < level; i++ {
			rank[i] = 0
			update[i] = zsl.header
			update[i].level[i].span = zsl.length
		}
		zsl.level = level
	}

	x = NewSkipNode(level, score, member)
	for i := range level {
		x.level[i].forward = update[i].level[i].forward
		update[i].level[i].forward = x

		x.level[i].span = update[i].level[i].span - (rank[0] - rank[i])
		update[i].level[i].span = (rank[0] - rank[i]) + 1
	}

	for i := level; i < zsl.level; i++ {
		update[i].level[i].span++
	}

	if update[0] == zsl.header {
		x.backward = nil
	} else {
		x.backward = update[0]
	}

	if x.level[0].forward != nil {
		x.level[0].forward.backward = x
	} else {
		zsl.tail = x
	}

	zsl.length++
	return x
}

func (zsl *SkipList) deleteNode(x *SkipNode, update []*SkipNode) {
	for i := 0; i < zsl.level; i++ {
		if update[i].level[i].forward == x {
			update[i].level[i].span += x.level[i].span - 1
			update[i].level[i].forward = x.level[i].forward
		} else {
			update[i].level[i].span--
		}
	}
	if x.level[0].forward != nil {
		x.level[0].forward.backward = x.backward
	} else {
		zsl.tail = x.backward
	}
	for zsl.level > 1 && zsl.header.level[zsl.level-1].forward == nil {
		zsl.level--
	}
	zsl.length--
}

func (zsl *SkipList) delete(score float64, member string) bool {
	update := make([]*SkipNode, zskipListMaxLevel)
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil &&
			(x.level[i].forward.score < score ||
				(x.level[i].forward.score == score && x.level[i].forward.member < member)) {
			x = x.level[i].forward
		}
		update[i] = x
	}
	x = x.level[0].forward
	if x != nil && score == x.score && x.member == member {
		zsl.deleteNode(x, update)
		return true
	}
	return false
}

func (zsl *SkipList) getRank(score float64, member string) uint64 {
	var rank uint64
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil &&
			(x.level[i].forward.score < score ||
				(x.level[i].forward.score == score && x.level[i].forward.member <= member)) {
			rank += x.level[i].span
			x = x.level[i].forward
		}
		if x != zsl.header && x.member == member {
			return rank
		}
	}
	return 0
}

func (zsl *SkipList) getElementByRank(rank uint64) *SkipNode {
	var traversed uint64
	x := zsl.header
	for i := zsl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && (traversed+x.level[i].span) <= rank {
			traversed += x.level[i].span
			x = x.level[i].forward
		}
		if traversed == rank {
			return x
		}
	}
	return nil
}
