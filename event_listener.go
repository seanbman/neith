package neith

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type OnEvent string

// DOM event types.
const (
	EventAbort              OnEvent = "abort"
	EventAnimationEnd       OnEvent = "animationend"
	EventAnimationIteration OnEvent = "animationiteration"
	EventAnimationStart     OnEvent = "animationstart"
	EventBlur               OnEvent = "blur"
	EventCanPlay            OnEvent = "canplay"
	EventCanPlayThrough     OnEvent = "canplaythrough"
	EventChange             OnEvent = "change"
	EventChangeCapture      OnEvent = "changecapture"
	EventClick              OnEvent = "click"
	EventCompositionEnd     OnEvent = "compositionend"
	EventCompositionStart   OnEvent = "compositionstart"
	EventCompositionUpdate  OnEvent = "compositionupdate"
	EventContextMenuCapture OnEvent = "contextmenucapture"
	EventCopy               OnEvent = "copy"
	EventCut                OnEvent = "cut"
	EventDoubleClickCapture OnEvent = "doubleclickcapture"
	EventDrag               OnEvent = "drag"
	EventDragEnd            OnEvent = "dragend"
	EventDragEnter          OnEvent = "dragenter"
	EventDragExitCapture    OnEvent = "dragexitcapture"
	EventDragLeave          OnEvent = "dragleave"
	EventDragOver           OnEvent = "dragover"
	EventDragStart          OnEvent = "dragstart"
	EventDrop               OnEvent = "drop"
	EventDurationChange     OnEvent = "durationchange"
	EventEmptied            OnEvent = "emptied"
	EventEncrypted          OnEvent = "encrypted"
	EventEnded              OnEvent = "ended"
	EventError              OnEvent = "error"
	EventFocus              OnEvent = "focus"
	EventGotPointerCapture  OnEvent = "gotpointercapture"
	EventInput              OnEvent = "input"
	EventInvalid            OnEvent = "invalid"
	EventKeyDown            OnEvent = "keydown"
	EventKeyPress           OnEvent = "keypress"
	EventKeyUp              OnEvent = "keyup"
	EventLoad               OnEvent = "load"
	EventLoadEnd            OnEvent = "loadend"
	EventLoadStart          OnEvent = "loadstart"
	EventLoadedData         OnEvent = "loadeddata"
	EventLoadedMetadata     OnEvent = "loadedmetadata"
	EventLostPointerCapture OnEvent = "lostpointercapture"
	EventMouseDown          OnEvent = "mousedown"
	EventMouseEnter         OnEvent = "mouseenter"
	EventMouseLeave         OnEvent = "mouseleave"
	EventMouseMove          OnEvent = "mousemove"
	EventMouseOut           OnEvent = "mouseout"
	EventMouseOver          OnEvent = "mouseover"
	EventMouseUp            OnEvent = "mouseup"
	EventPause              OnEvent = "pause"
	EventPlay               OnEvent = "play"
	EventPlaying            OnEvent = "playing"
	EventPointerCancel      OnEvent = "pointercancel"
	EventPointerDown        OnEvent = "pointerdown"
	EventPointerEnter       OnEvent = "pointerenter"
	EventPointerLeave       OnEvent = "pointerleave"
	EventPointerMove        OnEvent = "pointermove"
	EventPointerOut         OnEvent = "pointerout"
	EventPointerOver        OnEvent = "pointerover"
	EventPointerUp          OnEvent = "pointerup"
	EventProgress           OnEvent = "progress"
	EventRateChange         OnEvent = "ratechange"
	EventResetCapture       OnEvent = "resetcapture"
	EventScroll             OnEvent = "scroll"
	EventSeeked             OnEvent = "seeked"
	EventSeeking            OnEvent = "seeking"
	EventSelectCapture      OnEvent = "selectcapture"
	EventStalled            OnEvent = "stalled"
	EventSubmit             OnEvent = "submit"
	EventSuspend            OnEvent = "suspend"
	EventTimeUpdate         OnEvent = "timeupdate"
	EventToggle             OnEvent = "toggle"
	EventTouchCancel        OnEvent = "touchcancel"
	EventTouchEnd           OnEvent = "touchend"
	EventTouchMove          OnEvent = "touchmove"
	EventTouchStart         OnEvent = "touchstart"
	EventTransitionEnd      OnEvent = "transitionend"
	EventVolumeChange       OnEvent = "volumechange"
	EventWaiting            OnEvent = "waiting"
	EventWheel              OnEvent = "wheel"
)

// EventListener is the server-side handler metadata serialized into rendered
// components and echoed back by the browser when the matching DOM event fires.
type EventListener struct {
	context.Context `json:"-"`
	ID              string       `json:"id"`
	TargetID        string       `json:"target_id"`
	Handler         HandleFn     `json:"-"`
	On              OnEvent      `json:"on"`
	Data            any          `json:"data"`
	Uploads         []Upload     `json:"uploads,omitempty"`
	Submitter       *EventTarget `json:"submitter,omitempty"`
}

func newEventListener(on OnEvent, f FnComponent, h HandleFn) EventListener {
	if f.dispatch.conn == nil {
		f.dispatch.runtime().Config().Logger.Error("connection not found")
		return EventListener{}
	}

	el := EventListener{
		Context:  f.Context,
		ID:       uuid.New().String(),
		TargetID: f.id,
		Handler:  h,
		On:       on,
	}
	f.dispatch.runtime().eventListeners.Add(f.dispatch.conn, el)
	return el
}

type eventListeners struct {
	mu sync.Mutex
	el map[string]map[string]EventListener
}

func newEventListeners() eventListeners {
	return eventListeners{
		el: make(map[string]map[string]EventListener),
	}
}

func (e *eventListeners) Add(conn *conn, el EventListener) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.el[conn.ClientID]; !ok {
		e.el[conn.ClientID] = make(map[string]EventListener)
	}
	e.el[conn.ClientID][el.ID] = el
}

func (e *eventListeners) Delete(conn *conn) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.el, conn.ClientID)
}

func (e *eventListeners) Get(id string, conn *conn) (EventListener, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	event, ok := e.el[conn.ClientID][id]
	return event, ok
}
