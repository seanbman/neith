package fcmp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

var handlers = handlerPool{
	pool: make(map[string]handler),
}

type handlerPool struct {
	mu   sync.Mutex
	pool map[string]handler
}

func (h *handlerPool) Get(id string) (handler, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	handler, ok := h.pool[id]
	return handler, ok
}

func (h *handlerPool) Set(id string, handler handler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.pool[id] = handler
}

func (h *handlerPool) Delete(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.pool, id)
}

type HandleFn func(context.Context) FnComponent

type handler struct {
	http.Handler
	id        string
	in        chan Dispatch
	out       chan FnComponent
	handlesFn map[string]HandleFn
}

func newHandler() *handler {
	handler := handler{
		id:        uuid.New().String(),
		in:        make(chan Dispatch, 256),
		out:       make(chan FnComponent, 256),
		handlesFn: make(map[string]HandleFn),
	}
	handlers.Set(handler.id, handler)
	return &handler
}

func (h handler) ID() string {
	return h.id
}

func (h *handler) listen() {
	go func(h *handler) {
		for d := range h.in {
			switch d.Function {
			case ping:
				go h.Ping(d)
			case event:
				go h.Event(d)
			case custom:
				go h.CustomIn(d)
			case _error:
				go h.Error(d)
			default:
				d.FnError.Message = fmt.Sprintf(
					"function '%s' found, expected event or error on 'in' channel", d.Function)
				go h.Error(d)
			}
		}
	}(h)
	go func(h *handler) {
		for fn := range h.out {
			switch fn.dispatch.Function {
			case ping:
				go h.Ping(*fn.dispatch)
			case render:
				go h.Render(fn)
			case class:
				go h.Class(fn)
			case dom:
				go h.DOM(fn)
			case redirect:
				go h.Redirect(fn)
			case custom:
				go h.CustomOut(fn)
			case _error:
				go h.Error(*fn.dispatch)
			default:
				fn.dispatch.FnError.Message = fmt.Sprintf(
					"function '%s' found, expected event or error on 'in' channel", fn.dispatch.Function)
				go h.Error(*fn.dispatch)
			}
		}
	}(h)
}

func (h handler) Ping(d Dispatch) {
	if d.conn == nil {
		d.FnError.Message = "connection not found"
		h.Error(d)
		return
	}
	// Send ping to client
	if !d.FnPing.Client {
		d.FnPing.Server = true
		h.MarshalAndPublish(d)
		return
	}
}

func (h handler) Render(fn FnComponent) {
	// If there is no HTML to render, cancel dispatch
	if len(fn.dispatch.buf) == 0 && fn.dispatch.FnRender.HTML == "" && !fn.dispatch.FnRender.Remove {
		return
	}
	var data Writer
	fn.Render(context.Background(), &data)
	fn.dispatch.FnRender.HTML = sanitizeHTML(string(data.buf))
	h.MarshalAndPublish(*fn.dispatch)
}

func (h handler) Class(fn FnComponent) {
	// If there is no class to add, cancel dispatch
	if len(fn.dispatch.FnClass.Names) == 0 {
		return
	}
	h.MarshalAndPublish(*fn.dispatch)
}

func (h handler) DOM(fn FnComponent) {
	if fn.dispatch.FnDOM.TargetID == "" || fn.dispatch.FnDOM.Operation == "" {
		return
	}
	h.MarshalAndPublish(*fn.dispatch)
}

func (h handler) Redirect(fn FnComponent) {
	// If there is no URL to redirect to, cancel dispatch
	if fn.dispatch.FnRedirect.URL == "" {
		return
	}
	h.MarshalAndPublish(*fn.dispatch)
}

func (h handler) CustomIn(d Dispatch) {
	config.Logger.Debug("custom function in", d.FnCustom.Function+" result", d.FnCustom.Result)
}

func (h handler) CustomOut(fn FnComponent) {
	if fn.dispatch.FnCustom.Function == "" {
		return
	}
	h.MarshalAndPublish(*fn.dispatch)
}

func (h handler) MarshalAndPublish(d Dispatch) {
	if d.conn == nil {
		d.FnError.Message = "connection not found"
		h.Error(d)
		return
	}
	b, err := json.Marshal(d)
	if err != nil {
		d.FnError.Message = err.Error()
		h.Error(d)
		return
	}
	d.conn.Publish(b)
}

func (h handler) Event(d Dispatch) {
	if d.conn == nil {
		d.FnError.Message = ErrConnectionNotFound.Error()
		h.Error(d)
		return
	}
	listener, ok := evtListeners.Get(d.FnEvent.ID, d.conn)
	if !ok {
		d.FnError.Message = fmt.Sprintf("event listener with id '%s' not found", d.FnEvent.ID)
		h.Error(d)
		return
	}
	listener.Data = d.FnEvent.Data
	listener.Uploads = d.FnEvent.Uploads
	listener.Submitter = d.FnEvent.Submitter

	ctx := context.WithValue(listener.Context, EventKey, listener)
	response := listener.Handler(ctx)
	response.dispatch.conn = d.conn
	response.dispatch.HandlerID = d.HandlerID
	h.out <- response
}

func (h handler) Error(d Dispatch) {
	if config.Silent {
		return
	}
	config.Logger.Error(d.FnError)
}

type Writer struct {
	http.ResponseWriter
	buf []byte
}

func (w *Writer) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

func MiddleWareFn(h http.HandlerFunc, hf HandleFn) http.HandlerFunc {
	handler := newHandler()
	handler.listen()

	return func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("fcmp_id")
		if r.URL.Query().Get("fcmp_upload") == "1" {
			handler.Upload(w, r)
			return
		}
		if id == "" {
			writer := Writer{ResponseWriter: w}
			h(&writer, r)
			w.Write(writer.buf)
			return
		}
		newConnection, err := newConn(w, r, handler.id, id)
		if err != nil {
			config.Logger.Error(ErrConnectionFailed)
			config.Logger.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(ErrConnectionFailed))
			return
		}
		newConnection.HandlerID = handler.id

		ctx := context.WithValue(r.Context(), dispatchKey, dispatchDetails{
			ConnID:    id,
			Conn:      newConnection,
			HandlerID: handler.id,
		})
		ctx = context.WithValue(ctx, RequestKey, r)

		// Send initial fn to client
		fn := hf(ctx)
		fn.dispatch.conn = newConnection
		fn.dispatch.ConnID = id
		fn.dispatch.HandlerID = handler.id
		handler.out <- fn

		pinger := newDispatch(id)
		pinger.Function = ping
		pinger.FnPing.Server = true
		pinger.conn = newConnection
		pinger.ConnID = id
		pinger.HandlerID = handler.id

		// Send ping to client
		go func(d Dispatch) {
			for {
				// Check if connection is still open
				conn, _ := connPool.Get(d.ConnID)
				if conn != d.conn {
					// Connection has been replaced
					break
				}
				handler.Ping(d)
				time.Sleep(5 * time.Second)
			}
		}(*pinger)

		newConnection.listen()
	}
}
