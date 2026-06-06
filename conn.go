package neith

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var connPool = conns{
	pool: make(map[string]*conn),
}

func (c *conns) Get(id string) (*conn, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	conn, ok := c.pool[id]
	return conn, ok
}

func (c *conns) Set(id string, conn *conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pool[id] = conn
}

func (c *conns) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.pool, id)
}

type (
	conns struct {
		mu   sync.Mutex
		pool map[string]*conn
	}
	conn struct {
		websocket *websocket.Conn
		ID        string
		HandlerID string
		LastPing  time.Time
		Key       string
		Messages  chan []byte
	}
)

func newConn(w http.ResponseWriter, r *http.Request, handlerID string, ID string) (*conn, error) {
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
		ID:        ID,
		HandlerID: handlerID,
		Messages:  make(chan []byte, 16),
	}
	connPool.Set(c.ID, c)
	return c, nil
}

func (c *conn) close() error {
	if c == nil {
		return errors.New("cannot close nil connection")
	}
	go func() {
		// After CacheTimeOut, delete cache if connection is not re-established
		time.Sleep(config.CacheTimeOut)
		_, ok := connPool.Get(c.ID)
		if !ok {
			sm.delete(c.ID)
		}
	}()

	evtListeners.Delete(c)
	connPool.Delete(c.ID)
	c.websocket.Close()
	return nil
}

func (c *conn) listen() {
	go func(c *conn) {
		defer c.close()
		var dispatch Dispatch
		for {
			_, message, err := c.websocket.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(
					err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure,
					websocket.CloseNormalClosure,
				) {
					log.Printf("error: %v", err)
				}
				close(c.Messages)
				break
			}
			// Parse dispatch from websocket message
			err = json.Unmarshal(message, &dispatch)
			if err != nil {
				log.Printf("error: %v", err)
				continue
			}
			// Get handler from handler pool
			handler, ok := handlers.Get(dispatch.HandlerID)
			if !ok {
				log.Printf("error: handler '%s' not found", dispatch.HandlerID)
				continue
			}
			// Set conn on dispatch
			dispatch.conn = c
			// Dispatch to handler
			handler.in <- dispatch
		}
	}(c)

	for {
		msg, ok := <-c.Messages
		if !ok {
			c.close()
			break
		}
		if c.websocket == nil {
			break
		}

		if err := c.websocket.WriteMessage(1, msg); err != nil {
			config.Logger.Error("error writing message", "error", err)
			c.close()
		}
	}
}

func (c *conn) Publish(msg []byte) {
	// if msg is not json encodable, return
	_, err := json.Marshal(msg)
	if err != nil {
		config.Logger.Error("message not json encodable", "error", err)
		return
	}
	if c == nil {
		config.Logger.Warn("connection severed, message not sent")
		return
	}
	conn, _ := connPool.Get(c.ID)
	if conn != c {
		return
	}
	c.Messages <- msg
}

func (c *conn) Write(p []byte) (n int, err error) {
	c.Messages <- p
	return len(p), nil
}
