package neith

import (
	"testing"
	"time"
)

func TestConnCloseIsIdempotent(t *testing.T) {
	c := &conn{
		ClientID: "close-idempotent",
		done:     make(chan struct{}),
		Messages: make(chan []byte, 1),
	}
	clientSessions.Attach(c.ClientID, c)
	t.Cleanup(func() {
		clientSessions.Delete(c.ClientID)
	})

	if err := c.close(); err != nil {
		t.Fatalf("first close failed: %v", err)
	}
	if err := c.close(); err != nil {
		t.Fatalf("second close failed: %v", err)
	}

	select {
	case <-c.done:
	case <-time.After(time.Second):
		t.Fatal("connection close did not signal done")
	}
}

func TestConnCloseDoesNotDeleteReplacement(t *testing.T) {
	oldConn := &conn{
		ClientID: "replacement",
		done:     make(chan struct{}),
		Messages: make(chan []byte, 1),
	}
	newConn := &conn{
		ClientID: oldConn.ClientID,
		done:     make(chan struct{}),
		Messages: make(chan []byte, 1),
	}

	clientSessions.Attach(oldConn.ClientID, oldConn)
	clientSessions.Attach(newConn.ClientID, newConn)
	t.Cleanup(func() {
		clientSessions.Delete(newConn.ClientID)
	})

	if err := oldConn.close(); err != nil {
		t.Fatalf("old connection close failed: %v", err)
	}

	got, ok := clientSessions.ActiveConn(newConn.ClientID)
	if !ok {
		t.Fatal("replacement connection was deleted")
	}
	if got != newConn {
		t.Fatal("client session no longer points at replacement connection")
	}
}
