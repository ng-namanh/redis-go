package commands

import (
	"fmt"
	"net"
	"sync"

	"github.com/ng-namanh/redis-go/internal/resp"
)

var pubsubMutex sync.Mutex
var channels = make(map[string]map[net.Conn]struct{})

func countSubscriptions(conn net.Conn) int {
	count := 0
	for _, subs := range channels {
		if _, ok := subs[conn]; ok {
			count++
		}
	}
	return count
}

func getSubscriptions(conn net.Conn) []string {
	var subsList []string
	for ch, subs := range channels {
		if _, ok := subs[conn]; ok { // check if conn is subscribed to the channel
			subsList = append(subsList, ch)
		}
	}
	return subsList
}

func SUBSCRIBE(conn net.Conn, args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("wrong number of arguments for 'subscribe' command")
	}

	pubsubMutex.Lock()
	defer pubsubMutex.Unlock()

	var out []byte

	for _, ch := range args {
		if _, ok := channels[ch]; !ok {
			channels[ch] = make(map[net.Conn]struct{})
		}

		channels[ch][conn] = struct{}{}

		subCount := countSubscriptions(conn)

		reply := resp.WriteArray([]resp.RESP{
			{Type: resp.BulkString, Str: "subscribe"},
			{Type: resp.BulkString, Str: ch},
			{Type: resp.Integer, Int: int64(subCount)},
		})
		out = append(out, reply...)
	}

	return out, nil
}

func UNSUBSCRIBE(conn net.Conn, args []string) ([]byte, error) {
	pubsubMutex.Lock()
	defer pubsubMutex.Unlock()

	var out []byte

	channelsToRemove := args
	if len(args) == 0 {
		channelsToRemove = getSubscriptions(conn)
	}

	for _, ch := range channelsToRemove {
		if subs, ok := channels[ch]; ok {
			delete(subs, conn)
			if len(subs) == 0 {
				delete(channels, ch)
			}
		}

		subCount := countSubscriptions(conn)

		reply := resp.WriteArray([]resp.RESP{
			{Type: resp.BulkString, Str: "unsubscribe"},
			{Type: resp.BulkString, Str: ch},
			{Type: resp.Integer, Int: int64(subCount)},
		})
		out = append(out, reply...)
	}
	return out, nil
}

func PUBLISH(args []string) ([]byte, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("wrong number of arguments for 'publish' command")
	}

	ch := args[0]
	msg := args[1]

	pubsubMutex.Lock()

	count := 0

	subs, ok := channels[ch]

	if ok {
		count = len(subs)
		reply := resp.WriteArray([]resp.RESP{
			{Type: resp.BulkString, Str: "message"},
			{Type: resp.BulkString, Str: ch},
			{Type: resp.BulkString, Str: msg},
		})

		for conn := range subs {
			go func(conn net.Conn) {
				conn.Write(reply)
			}(conn)
		}
	}

	pubsubMutex.Unlock()

	Propagate("PUBLISH", args)

	return resp.WriteInteger(int64(count)), nil
}

func RemoveSubscriber(conn net.Conn) {
	pubsubMutex.Lock()
	defer pubsubMutex.Unlock()

	for ch, subs := range channels {
		if _, ok := subs[conn]; ok {
			delete(subs, conn)
			if len(subs) == 0 {
				delete(channels, ch)
			}
		}
	}
}
