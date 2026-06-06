package neith

import "sync"

// clientSession represents one browser client visiting a Neith app.
//
// A session can outlive a single websocket connection. Refreshes, reconnects,
// and network drops replace activeConn while preserving the session ID used for
// cache and event scoping.
type clientSession struct {
	ID         string
	activeConn *conn
}

type clientSessionRegistry struct {
	mu       sync.Mutex
	sessions map[string]*clientSession
}

func newClientSessionRegistry() clientSessionRegistry {
	return clientSessionRegistry{
		sessions: make(map[string]*clientSession),
	}
}

func (r *clientSessionRegistry) Attach(clientID string, conn *conn) (*clientSession, *conn) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[clientID]
	if !ok {
		session = &clientSession{ID: clientID}
		r.sessions[clientID] = session
	}
	previous := session.activeConn
	session.activeConn = conn
	return session, previous
}

func (r *clientSessionRegistry) ActiveConn(clientID string) (*conn, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[clientID]
	if !ok || session.activeConn == nil {
		return nil, false
	}
	return session.activeConn, true
}

func (r *clientSessionRegistry) Detach(clientID string, conn *conn) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[clientID]
	if ok && session.activeConn == conn {
		session.activeConn = nil
	}
}

func (r *clientSessionRegistry) DeleteIfInactive(clientID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessions[clientID]
	if !ok || session.activeConn != nil {
		return false
	}
	delete(r.sessions, clientID)
	return true
}

func (r *clientSessionRegistry) Delete(clientID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, clientID)
}
