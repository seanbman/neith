package neith

import (
	"testing"
	"time"
)

func TestConnCloseIsIdempotent(t *testing.T) {
	rt := newRuntime(config)
	c := &conn{
		rt:       rt,
		ClientID: "close-idempotent",
		done:     make(chan struct{}),
		Messages: make(chan []byte, 1),
	}
	rt.sessions.Attach(c.ClientID, c)
	t.Cleanup(func() {
		rt.sessions.Delete(c.ClientID)
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
	rt := newRuntime(config)
	oldConn := &conn{
		rt:       rt,
		ClientID: "replacement",
		done:     make(chan struct{}),
		Messages: make(chan []byte, 1),
	}
	newConn := &conn{
		rt:       rt,
		ClientID: oldConn.ClientID,
		done:     make(chan struct{}),
		Messages: make(chan []byte, 1),
	}

	rt.sessions.Attach(oldConn.ClientID, oldConn)
	rt.sessions.Attach(newConn.ClientID, newConn)
	t.Cleanup(func() {
		rt.sessions.Delete(newConn.ClientID)
	})

	if err := oldConn.close(); err != nil {
		t.Fatalf("old connection close failed: %v", err)
	}

	got, ok := rt.sessions.ActiveConn(newConn.ClientID)
	if !ok {
		t.Fatal("replacement connection was deleted")
	}
	if got != newConn {
		t.Fatal("client session no longer points at replacement connection")
	}
}
