package neith

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type OnEvent string

// DOM event types.
const (
	OnAbort              OnEvent = "abort"
	OnAnimationEnd       OnEvent = "animationend"
	OnAnimationIteration OnEvent = "animationiteration"
	OnAnimationStart     OnEvent = "animationstart"
	OnBlur               OnEvent = "blur"
	OnCanPlay            OnEvent = "canplay"
	OnCanPlayThrough     OnEvent = "canplaythrough"
	OnChange             OnEvent = "change"
	OnChangeCapture      OnEvent = "changecapture"
	OnClick              OnEvent = "click"
	OnCompositionEnd     OnEvent = "compositionend"
	OnCompositionStart   OnEvent = "compositionstart"
	OnCompositionUpdate  OnEvent = "compositionupdate"
	OnContextMenuCapture OnEvent = "contextmenucapture"
	OnCopy               OnEvent = "copy"
	OnCut                OnEvent = "cut"
	OnDoubleClickCapture OnEvent = "doubleclickcapture"
	OnDrag               OnEvent = "drag"
	OnDragEnd            OnEvent = "dragend"
	OnDragEnter          OnEvent = "dragenter"
	OnDragExitCapture    OnEvent = "dragexitcapture"
	OnDragLeave          OnEvent = "dragleave"
	OnDragOver           OnEvent = "dragover"
	OnDragStart          OnEvent = "dragstart"
	OnDrop               OnEvent = "drop"
	OnDurationChange     OnEvent = "durationchange"
	OnEmptied            OnEvent = "emptied"
	OnEncrypted          OnEvent = "encrypted"
	OnEnded              OnEvent = "ended"
	OnError              OnEvent = "error"
	OnFocus              OnEvent = "focus"
	OnGotPointerCapture  OnEvent = "gotpointercapture"
	OnInput              OnEvent = "input"
	OnInvalid            OnEvent = "invalid"
	OnKeyDown            OnEvent = "keydown"
	OnKeyPress           OnEvent = "keypress"
	OnKeyUp              OnEvent = "keyup"
	OnLoad               OnEvent = "load"
	OnLoadEnd            OnEvent = "loadend"
	OnLoadStart          OnEvent = "loadstart"
	OnLoadedData         OnEvent = "loadeddata"
	OnLoadedMetadata     OnEvent = "loadedmetadata"
	OnLostPointerCapture OnEvent = "lostpointercapture"
	OnMouseDown          OnEvent = "mousedown"
	OnMouseEnter         OnEvent = "mouseenter"
	OnMouseLeave         OnEvent = "mouseleave"
	OnMouseMove          OnEvent = "mousemove"
	OnMouseOut           OnEvent = "mouseout"
	OnMouseOver          OnEvent = "mouseover"
	OnMouseUp            OnEvent = "mouseup"
	OnPause              OnEvent = "pause"
	OnPlay               OnEvent = "play"
	OnPlaying            OnEvent = "playing"
	OnPointerCancel      OnEvent = "pointercancel"
	OnPointerDown        OnEvent = "pointerdown"
	OnPointerEnter       OnEvent = "pointerenter"
	OnPointerLeave       OnEvent = "pointerleave"
	OnPointerMove        OnEvent = "pointermove"
	OnPointerOut         OnEvent = "pointerout"
	OnPointerOver        OnEvent = "pointerover"
	OnPointerUp          OnEvent = "pointerup"
	OnProgress           OnEvent = "progress"
	OnRateChange         OnEvent = "ratechange"
	OnResetCapture       OnEvent = "resetcapture"
	OnScroll             OnEvent = "scroll"
	OnSeeked             OnEvent = "seeked"
	OnSeeking            OnEvent = "seeking"
	OnSelectCapture      OnEvent = "selectcapture"
	OnStalled            OnEvent = "stalled"
	OnSubmit             OnEvent = "submit"
	OnSuspend            OnEvent = "suspend"
	OnTimeUpdate         OnEvent = "timeupdate"
	OnToggle             OnEvent = "toggle"
	OnTouchCancel        OnEvent = "touchcancel"
	OnTouchEnd           OnEvent = "touchend"
	OnTouchMove          OnEvent = "touchmove"
	OnTouchStart         OnEvent = "touchstart"
	OnTransitionEnd      OnEvent = "transitionend"
	OnVolumeChange       OnEvent = "volumechange"
	OnWaiting            OnEvent = "waiting"
	OnWheel              OnEvent = "wheel"
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
		config.Logger.Error("connection not found")
	}

	el := EventListener{
		Context:  f.Context,
		ID:       uuid.New().String(),
		TargetID: f.id,
		Handler:  h,
		On:       on,
	}
	evtListeners.Add(f.dispatch.conn, el)
	return el
}

type eventListeners struct {
	mu sync.Mutex
	el map[string]map[string]EventListener
}

var evtListeners = eventListeners{
	el: make(map[string]map[string]EventListener),
}

func (e *eventListeners) Add(conn *conn, el EventListener) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.el[conn.ID]; !ok {
		e.el[conn.ID] = make(map[string]EventListener)
	}
	e.el[conn.ID][el.ID] = el
}

func (e *eventListeners) Delete(conn *conn) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.el, conn.ID)
}

func (e *eventListeners) Get(id string, conn *conn) (EventListener, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()

	event, ok := e.el[conn.ID][id]
	return event, ok
}
