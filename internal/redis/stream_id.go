package redis

import (
	"errors"
	"strconv"
	"strings"
)

const (
	ErrXADDIDNotGreaterThanTop = "ERR The ID specified in XADD is equal or smaller than the target stream top item"
	ErrXADDIDMustBeGreater0    = "ERR The ID specified in XADD must be greater than 0-0"
)

// ErrNotGreaterThanTop means the proposed ID is not strictly greater than the stream's last entry.
var ErrNotGreaterThanTop = errors.New("stream id not greater than top")

// StreamId is a Redis stream entry id: <milliseconds>-<sequence>.
type StreamId struct {
	Ms  uint64
	Seq uint64
}

// GreaterThan reports whether a is strictly greater than b (Redis ordering).
func (a StreamId) GreaterThan(b StreamId) bool {
	return a.Ms > b.Ms || (a.Ms == b.Ms && a.Seq > b.Seq)
}

// Splits an ID into milliseconds and sequence parts from the "<ms>-<seq>" format
func splitMsSeq(id string) (msStr, seqStr string, ok bool) {
	i := strings.IndexByte(id, '-')
	if i <= 0 || i == len(id)-1 {
		return "", "", false
	}
	return id[:i], id[i+1:], true
}

func LastStreamEntryID(s *Stream) string {
	if s == nil || len(s.entries) == 0 {
		return ""
	}
	return s.entries[len(s.entries)-1].id
}

// ParseXRANGEBound parses an XRANGE start or end ID: full "<ms>-<seq>", or a
// milliseconds-only token. For ms-only, start uses sequence 0; end uses the
// maximum sequence so all entries in that millisecond are included.
func ParseXRANGEBound(s string, isStart bool) (StreamId, bool) {
	// Special IDs per XRANGE docs: "-" is minimal possible ID, "+" is maximal.
	// Using 0-0 and maxUint64-maxUint64 ensures closed-interval comparisons work.
	if s == "-" {
		return StreamId{Ms: 0, Seq: 0}, true
	}
	if s == "+" {
		return StreamId{Ms: ^uint64(0), Seq: ^uint64(0)}, true
	}

	if strings.Contains(s, "-") {
		return ParseStreamID(s)
	}
	ms, err := strconv.ParseUint(s, 10, 64)
	if err != nil || s == "" {
		return StreamId{}, false
	}
	if isStart {
		return StreamId{Ms: ms, Seq: 0}, true
	}
	return StreamId{Ms: ms, Seq: ^uint64(0)}, true
}

// StreamIdGte reports whether a >= b in Redis stream ID order (inclusive).
func StreamIdGte(a, b StreamId) bool {
	return a.Ms > b.Ms || (a.Ms == b.Ms && a.Seq >= b.Seq)
}

// StreamIdLte reports whether a <= b in Redis stream ID order (inclusive).
func StreamIdLte(a, b StreamId) bool {
	return a.Ms < b.Ms || (a.Ms == b.Ms && a.Seq <= b.Seq)
}

// ParseStreamID parses an explicit "<ms>-<seq>" id (single '-', decimal parts).
func ParseStreamID(id string) (StreamId, bool) {
	msStr, seqStr, ok := splitMsSeq(id)
	if !ok || strings.Contains(seqStr, "-") { // only one '-' separator
		return StreamId{}, false
	}
	ms, err1 := strconv.ParseUint(msStr, 10, 64)
	seq, err2 := strconv.ParseUint(seqStr, 10, 64)
	if err1 != nil || err2 != nil {
		return StreamId{}, false
	}
	return StreamId{Ms: ms, Seq: seq}, true
}

// FormatStreamID renders "<ms>-<seq>".
func FormatStreamID(ms, seq uint64) string {
	return strconv.FormatUint(ms, 10) + "-" + strconv.FormatUint(seq, 10)
}

// NextAutoFull returns the ID for XADD when the client passes "*".
// nowMillis is usually time.Now().UnixMilli(); lastEntryID is "" if the stream is empty.
func NextAutoFull(nowMillis uint64, lastEntryID string) string {
	if lastEntryID == "" {
		return FormatStreamID(nowMillis, 0)
	}
	last, ok := ParseStreamID(lastEntryID)
	if !ok {
		return FormatStreamID(nowMillis, 0)
	}
	var ms, seq uint64
	switch {
	case nowMillis > last.Ms:
		ms, seq = nowMillis, 0
	case nowMillis == last.Ms:
		ms, seq = nowMillis, last.Seq+1
	default:
		ms, seq = last.Ms, last.Seq+1
	}
	return FormatStreamID(ms, seq)
}

// ParsePartialSeqAutoID reports whether id is "<ms>-*" (sequence auto-generated).
func ParsePartialSeqAutoID(id string) (ms uint64, ok bool) {
	msStr, seqStr, ok := splitMsSeq(id)
	if !ok || seqStr != "*" || strings.Contains(msStr, "-") {
		return 0, false
	}
	m, err := strconv.ParseUint(msStr, 10, 64)
	if err != nil {
		return 0, false
	}
	return m, true
}

// NextPartialSeqStreamID builds the ID for XADD when the client passes "<ms>-*".
// Pass lastEntryID == "" for an empty stream.
func NextPartialSeqStreamID(ms uint64, lastEntryID string) (finalID string, err error) {
	if lastEntryID == "" {
		if ms == 0 {
			return FormatStreamID(0, 1), nil
		}
		return FormatStreamID(ms, 0), nil
	}
	last, ok := ParseStreamID(lastEntryID)
	if !ok {
		if ms == 0 {
			return FormatStreamID(0, 1), nil
		}
		return FormatStreamID(ms, 0), nil
	}
	if ms < last.Ms {
		return "", ErrNotGreaterThanTop
	}
	if ms > last.Ms {
		return FormatStreamID(ms, 0), nil
	}
	return FormatStreamID(ms, last.Seq+1), nil
}
