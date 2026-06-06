package neith

import (
	"testing"
	"time"
)

func TestConnCloseIsIdempotent(t *testing.T) {
	c := &conn{
		ID:       "close-idempotent",
		done:     make(chan struct{}),
		Messages: make(chan []byte, 1),
	}
	connPool.Set(c.ID, c)
	t.Cleanup(func() {
		connPool.Delete(c.ID)
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
		ID:       "replacement",
		done:     make(chan struct{}),
		Messages: make(chan []byte, 1),
	}
	newConn := &conn{
		ID:       oldConn.ID,
		done:     make(chan struct{}),
		Messages: make(chan []byte, 1),
	}

	connPool.Set(oldConn.ID, oldConn)
	connPool.Set(newConn.ID, newConn)
	t.Cleanup(func() {
		connPool.Delete(newConn.ID)
	})

	if err := oldConn.close(); err != nil {
		t.Fatalf("old connection close failed: %v", err)
	}

	got, ok := connPool.Get(newConn.ID)
	if !ok {
		t.Fatal("replacement connection was deleted")
	}
	if got != newConn {
		t.Fatal("connection pool no longer points at replacement")
	}
}
