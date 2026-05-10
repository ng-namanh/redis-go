package redis_test

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/ng-namanh/redis-go/internal/commands"
)

type MockConn struct {
	net.Conn
	writeBuffer *bytes.Buffer
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	return m.writeBuffer.Write(b)
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *MockConn) Close() error {
	return nil
}

func TestPubSub(t *testing.T) {
	commands.ResetForTesting()

	conn1 := &MockConn{writeBuffer: new(bytes.Buffer)}
	conn2 := &MockConn{writeBuffer: new(bytes.Buffer)}

	t.Run("SUBSCRIBE", func(t *testing.T) {
		res, err := commands.SUBSCRIBE(conn1, []string{"news", "sports"})
		if err != nil {
			t.Fatalf("SUBSCRIBE failed: %v", err)
		}

		if !bytes.Contains(res, []byte("subscribe")) || !bytes.Contains(res, []byte("news")) {
			t.Errorf("Expected subscription confirmation for news, got %q", res)
		}
	})

	t.Run("PUBLISH", func(t *testing.T) {
		// conn2 subscribes to sports
		_, _ = commands.SUBSCRIBE(conn2, []string{"sports"})

		// conn1 clears its buffer
		conn1.writeBuffer.Reset()
		conn2.writeBuffer.Reset()

		// Publish to news
		res, err := commands.PUBLISH([]string{"news", "breaking news!"})
		if err != nil {
			t.Fatalf("PUBLISH failed: %v", err)
		}
		if !bytes.Contains(res, []byte(":1")) {
			t.Errorf("Expected 1 receiver for 'news', got %q", res)
		}

		// Wait for async write
		time.Sleep(10 * time.Millisecond)

		if !bytes.Contains(conn1.writeBuffer.Bytes(), []byte("message")) || !bytes.Contains(conn1.writeBuffer.Bytes(), []byte("breaking news!")) {
			t.Errorf("conn1 did not receive the published message, got %q", conn1.writeBuffer.Bytes())
		}
		if conn2.writeBuffer.Len() > 0 {
			t.Errorf("conn2 received message but was not subscribed to 'news'")
		}
	})

	t.Run("UNSUBSCRIBE", func(t *testing.T) {
		res, err := commands.UNSUBSCRIBE(conn1, []string{"news"})
		if err != nil {
			t.Fatalf("UNSUBSCRIBE failed: %v", err)
		}

		if !bytes.Contains(res, []byte("unsubscribe")) || !bytes.Contains(res, []byte("news")) {
			t.Errorf("Expected unsubscription confirmation for news, got %q", res)
		}

		conn1.writeBuffer.Reset()
		commands.PUBLISH([]string{"news", "another news!"})

		time.Sleep(10 * time.Millisecond)

		if conn1.writeBuffer.Len() > 0 {
			t.Errorf("conn1 received message after unsubscribing")
		}
	})

	t.Run("RemoveSubscriber", func(t *testing.T) {
		commands.RemoveSubscriber(conn2)

		conn2.writeBuffer.Reset()
		commands.PUBLISH([]string{"sports", "goal!"})

		time.Sleep(10 * time.Millisecond)

		if conn2.writeBuffer.Len() > 0 {
			t.Errorf("conn2 received message after being removed")
		}
	})
}
