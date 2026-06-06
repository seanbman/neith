package fcmp

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

type OnEvent string

// DOM event types
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
	id := uuid.New().String()
	el := EventListener{
		Context:  f.Context,
		ID:       id,
		TargetID: f.id,
		Handler:  h,
		On:       on,
	}
	evtListeners.Add(f.dispatch.conn, el)
	return el
}

// Store and retrieve event listeners
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

// EventData unmarshals event listener data T from the client
func EventData[T any](ctx context.Context) (T, error) {
	e, ok := ctx.Value(EventKey).(EventListener)
	if !ok {
		return *new(T), ErrCtxMissingEvent
	}
	var t T
	b, err := json.Marshal(e.Data)
	if err != nil {
		return t, err
	}
	err = json.Unmarshal(b, &t)
	return t, err
}

// EventUploads returns file metadata uploaded before the current event dispatch.
//
// File bytes are posted to fcmp's upload endpoint over HTTP. EventData still
// contains normal form values, and EventUploads exposes the uploaded files.
func EventUploads(ctx context.Context) ([]Upload, error) {
	e, ok := ctx.Value(EventKey).(EventListener)
	if !ok {
		return nil, ErrCtxMissingEvent
	}
	return append([]Upload(nil), e.Uploads...), nil
}

// EventSubmitter returns the button or input that submitted a form event.
func EventSubmitter(ctx context.Context) (*EventTarget, error) {
	e, ok := ctx.Value(EventKey).(EventListener)
	if !ok {
		return nil, ErrCtxMissingEvent
	}
	if e.Submitter == nil {
		return nil, nil
	}
	submitter := *e.Submitter
	return &submitter, nil
}

// Upload describes one file uploaded for an event.
type Upload struct {
	ID          string `json:"id"`
	FieldName   string `json:"field_name"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Path        string `json:"path"`
}

// Event data types
type EventTarget struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	ClassList       []string `json:"classList"`
	TagName         string   `json:"tagName"`
	InnerHTML       string   `json:"innerHTML"`
	OuterHTML       string   `json:"outerHTML"`
	Value           string   `json:"value"`
	Checked         bool     `json:"checked"`
	Disabled        bool     `json:"disabled"`
	Hidden          bool     `json:"hidden"`
	Style           string   `json:"style"`
	Attributes      []string `json:"attributes"`
	Dataset         []string `json:"dataset"`
	SelectedOptions []string `json:"selectedOptions"`
}

type PointerEvent struct {
	IsTrusted        bool        `json:"isTrusted"`
	AltKey           bool        `json:"altKey"`
	Bubbles          bool        `json:"bubbles"`
	Button           int         `json:"button"`
	Buttons          int         `json:"buttons"`
	Cancelable       bool        `json:"cancelable"`
	ClientX          int         `json:"clientX"`
	ClientY          int         `json:"clientY"`
	Composed         bool        `json:"composed"`
	CtrlKey          bool        `json:"ctrlKey"`
	CurrentTarget    EventTarget `json:"currentTarget"`
	DefaultPrevented bool        `json:"defaultPrevented"`
	Detail           int         `json:"detail"`
	EventPhase       int         `json:"eventPhase"`
	Height           int         `json:"height"`
	IsPrimary        bool        `json:"isPrimary"`
	MetaKey          bool        `json:"metaKey"`
	MovementX        int         `json:"movementX"`
	MovementY        int         `json:"movementY"`
	OffsetX          int         `json:"offsetX"`
	OffsetY          int         `json:"offsetY"`
	PageX            int         `json:"pageX"`
	PageY            int         `json:"pageY"`
	PointerId        int         `json:"pointerId"`
	PointerType      string      `json:"pointerType"`
	Pressure         int         `json:"pressure"`
	RelatedTarget    EventTarget `json:"relatedTarget"`
	Target           EventTarget `json:"target"`
}

type TouchEvent struct {
	ChangedTouches []Touch `json:"changedTouches"`
	TargetTouches  []Touch `json:"targetTouches"`
	Touches        []Touch `json:"touches"`
	LayerX         int     `json:"layerX"`
	LayerY         int     `json:"layerY"`
	PageX          int     `json:"pageX"`
	PageY          int     `json:"pageY"`
}

type Touch struct {
	ClientX       int         `json:"clientX"`
	ClientY       int         `json:"clientY"`
	Identifier    int         `json:"identifier"`
	PageX         int         `json:"pageX"`
	PageY         int         `json:"pageY"`
	RadiusX       float64     `json:"radiusX"`
	RadiusY       float64     `json:"radiusY"`
	RotationAngle int         `json:"rotationAngle"`
	ScreenX       int         `json:"screenX"`
	ScreenY       int         `json:"screenY"`
	Target        EventTarget `json:"target"`
}

type DragEvent struct {
	IsTrusted        bool        `json:"isTrusted"`
	AltKey           bool        `json:"altKey"`
	Bubbles          bool        `json:"bubbles"`
	Button           int         `json:"button"`
	Buttons          int         `json:"buttons"`
	Cancelable       bool        `json:"cancelable"`
	ClientX          int         `json:"clientX"`
	ClientY          int         `json:"clientY"`
	Composed         bool        `json:"composed"`
	CtrlKey          bool        `json:"ctrlKey"`
	CurrentTarget    EventTarget `json:"currentTarget"`
	DefaultPrevented bool        `json:"defaultPrevented"`
	Detail           int         `json:"detail"`
	EventPhase       int         `json:"eventPhase"`
	MetaKey          bool        `json:"metaKey"`
	MovementX        int         `json:"movementX"`
	MovementY        int         `json:"movementY"`
	OffsetX          int         `json:"offsetX"`
	OffsetY          int         `json:"offsetY"`
	PageX            int         `json:"pageX"`
	PageY            int         `json:"pageY"`
	RelatedTarget    EventTarget `json:"relatedTarget"`
	Target           EventTarget `json:"target"`
}

type MouseEvent struct {
	IsTrusted        bool        `json:"isTrusted"`
	AltKey           bool        `json:"altKey"`
	Bubbles          bool        `json:"bubbles"`
	Button           int         `json:"button"`
	Buttons          int         `json:"buttons"`
	Cancelable       bool        `json:"cancelable"`
	ClientX          int         `json:"clientX"`
	ClientY          int         `json:"clientY"`
	Composed         bool        `json:"composed"`
	CtrlKey          bool        `json:"ctrlKey"`
	CurrentTarget    EventTarget `json:"currentTarget"`
	DefaultPrevented bool        `json:"defaultPrevented"`
	Detail           int         `json:"detail"`
	EventPhase       int         `json:"eventPhase"`
	MetaKey          bool        `json:"metaKey"`
	MovementX        int         `json:"movementX"`
	MovementY        int         `json:"movementY"`
	OffsetX          int         `json:"offsetX"`
	OffsetY          int         `json:"offsetY"`
	PageX            int         `json:"pageX"`
	PageY            int         `json:"pageY"`
	RelatedTarget    EventTarget `json:"relatedTarget"`
	Target           EventTarget `json:"target"`
}

type KeyboardEvent struct {
	IsTrusted        bool        `json:"isTrusted"`
	AltKey           bool        `json:"altKey"`
	Bubbles          bool        `json:"bubbles"`
	Cancelable       bool        `json:"cancelable"`
	Code             string      `json:"code"`
	Composed         bool        `json:"composed"`
	CtrlKey          bool        `json:"ctrlKey"`
	CurrentTarget    EventTarget `json:"currentTarget"`
	DefaultPrevented bool        `json:"defaultPrevented"`
	Detail           int         `json:"detail"`
	EventPhase       int         `json:"eventPhase"`
	IsComposing      bool        `json:"isComposing"`
	Key              string      `json:"key"`
	Location         int         `json:"location"`
	MetaKey          bool        `json:"metaKey"`
	Repeat           bool        `json:"repeat"`
	ShiftKey         bool        `json:"shiftKey"`
	Target           EventTarget `json:"target"`
}

type FormDataEvent struct {
	IsTrusted        bool           `json:"isTrusted"`
	Bubbles          bool           `json:"bubbles"`
	Cancelable       bool           `json:"cancelable"`
	Composed         bool           `json:"composed"`
	CurrentTarget    EventTarget    `json:"currentTarget"`
	DefaultPrevented bool           `json:"defaultPrevented"`
	EventPhase       int            `json:"eventPhase"`
	FormData         map[string]any `json:"formData"`
}
