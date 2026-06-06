package neith

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type (
	conn struct {
		websocket *websocket.Conn
		session   *clientSession
		closeOnce sync.Once
		done      chan struct{}
		ClientID  string
		HandlerID string
		LastPing  time.Time
		Key       string
		Messages  chan []byte
	}
)

func newConn(w http.ResponseWriter, r *http.Request, handlerID string, clientID string) (*conn, error) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	websocket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, errors.New("failed to upgrade connection")
	}

	c := &conn{
		websocket: websocket,
		done:      make(chan struct{}),
		ClientID:  clientID,
		HandlerID: handlerID,
		Messages:  make(chan []byte, 16),
	}
	session, previous := clientSessions.Attach(clientID, c)
	c.session = session
	if previous != nil && previous != c {
		_ = previous.close()
	}
	return c, nil
}

func (c *conn) close() error {
	if c == nil {
		return errors.New("cannot close nil connection")
	}

	c.closeOnce.Do(func() {
		if c.done != nil {
			close(c.done)
		}

		if c.ClientID != "" {
			evtListeners.Delete(c)
			clientSessions.Detach(c.ClientID, c)
			go c.cleanupCacheAfterTimeout()
		}

		if c.websocket != nil {
			if err := c.websocket.Close(); err != nil {
				config.Logger.Debug("error closing websocket", "error", err)
			}
		}
	})
	return nil
}

func (c *conn) cleanupCacheAfterTimeout() {
	time.Sleep(config.CacheTimeOut)
	if clientSessions.DeleteIfInactive(c.ClientID) {
		sm.delete(c.ClientID)
	}
}

func (c *conn) readLoop() {
	defer c.close()

	for {
		_, message, err := c.websocket.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
				websocket.CloseNormalClosure,
			) {
				config.Logger.Error("error reading websocket message", "error", err)
			}
			return
		}

		var dispatch Dispatch
		if err := json.Unmarshal(message, &dispatch); err != nil {
			config.Logger.Error("error decoding websocket dispatch", "error", err)
			continue
		}

		handler, ok := handlers.Get(dispatch.HandlerID)
		if !ok {
			config.Logger.Error("handler not found", "handler_id", dispatch.HandlerID)
			continue
		}

		dispatch.conn = c
		select {
		case <-c.done:
			return
		case handler.in <- dispatch:
		}
	}
}

func (c *conn) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case msg, ok := <-c.Messages:
			if !ok {
				c.close()
				return
			}
			if c.websocket == nil {
				c.close()
				return
			}
			if err := c.websocket.WriteMessage(1, msg); err != nil {
				config.Logger.Error("error writing message", "error", err)
				c.close()
				return
			}
		}
	}
}

func (c *conn) listen() {
	go c.readLoop()
	c.writeLoop()
}

func (c *conn) Publish(msg []byte) {
	if c == nil {
		config.Logger.Warn("connection severed, message not sent")
		return
	}
	if c.Messages == nil {
		config.Logger.Warn("connection has no message queue, message not sent")
		return
	}
	if _, err := json.Marshal(msg); err != nil {
		config.Logger.Error("message not json encodable", "error", err)
		return
	}

	conn, _ := clientSessions.ActiveConn(c.ClientID)
	if conn != c {
		return
	}
	if c.done == nil {
		c.Messages <- msg
		return
	}

	select {
	case <-c.done:
		return
	case c.Messages <- msg:
	}
}

func (c *conn) Write(p []byte) (n int, err error) {
	if c == nil || c.Messages == nil {
		return 0, errors.New("connection not writable")
	}
	if c.done == nil {
		c.Messages <- p
		return len(p), nil
	}

	select {
	case <-c.done:
		return 0, errors.New("connection closed")
	case c.Messages <- p:
		return len(p), nil
	}
}
