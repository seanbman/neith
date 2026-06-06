package neith

import "testing"

func TestClientSessionAttachReplacesActiveConnection(t *testing.T) {
	registry := newClientSessionRegistry()
	oldConn := &conn{ClientID: "client"}
	newConn := &conn{ClientID: "client"}

	session, previous := registry.Attach(oldConn.ClientID, oldConn)
	if session.ID != oldConn.ClientID {
		t.Fatalf("expected session ID %q, got %q", oldConn.ClientID, session.ID)
	}
	if previous != nil {
		t.Fatal("first attached connection should not replace anything")
	}

	session, previous = registry.Attach(newConn.ClientID, newConn)
	if previous != oldConn {
		t.Fatal("expected second attach to return previous active connection")
	}
	if session.activeConn != newConn {
		t.Fatal("expected new connection to become active")
	}

	registry.Detach(oldConn.ClientID, oldConn)
	active, ok := registry.ActiveConn(newConn.ClientID)
	if !ok {
		t.Fatal("new connection should still be active after stale detach")
	}
	if active != newConn {
		t.Fatal("stale detach changed the active connection")
	}
}
