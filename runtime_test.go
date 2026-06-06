package neith

import (
	"context"
	"testing"
)

func TestRuntimeIsolatesCacheStores(t *testing.T) {
	rtA := newRuntime(config)
	rtB := newRuntime(config)
	ctxA := testRuntimeContext(rtA, "client")
	ctxB := testRuntimeContext(rtB, "client")

	cacheA, err := NewCache(ctxA, "count", 1)
	if err != nil {
		t.Fatalf("create cache A: %v", err)
	}
	cacheB, err := NewCache(ctxB, "count", 10)
	if err != nil {
		t.Fatalf("create cache B: %v", err)
	}

	if err := cacheA.Set(2); err != nil {
		t.Fatalf("set cache A: %v", err)
	}

	if got := cacheA.Value(); got != 2 {
		t.Fatalf("expected cache A value 2, got %d", got)
	}
	if got := cacheB.Value(); got != 10 {
		t.Fatalf("expected cache B value 10, got %d", got)
	}
}

func TestRuntimeIsolatesClientSessions(t *testing.T) {
	rtA := newRuntime(config)
	rtB := newRuntime(config)
	connA := &conn{rt: rtA, ClientID: "client"}
	connB := &conn{rt: rtB, ClientID: "client"}

	rtA.sessions.Attach(connA.ClientID, connA)
	rtB.sessions.Attach(connB.ClientID, connB)

	activeA, ok := rtA.sessions.ActiveConn("client")
	if !ok || activeA != connA {
		t.Fatal("runtime A should keep its own active connection")
	}

	activeB, ok := rtB.sessions.ActiveConn("client")
	if !ok || activeB != connB {
		t.Fatal("runtime B should keep its own active connection")
	}
}

func testRuntimeContext(rt *runtime, clientID string) context.Context {
	dd := dispatchDetails{
		Runtime:   rt,
		ClientID:  clientID,
		Conn:      &conn{rt: rt, ClientID: clientID},
		HandlerID: "handler",
	}
	return context.WithValue(context.Background(), dispatchKey, dd)
}
