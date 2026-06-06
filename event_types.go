package fcmp

// Upload describes one file uploaded for an event.
type Upload struct {
	ID          string `json:"id"`
	FieldName   string `json:"field_name"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Path        string `json:"path"`
}

// EventTarget is a JSON-safe snapshot of a DOM element involved in an event.
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

// PointerEvent contains the pointer payload sent by the browser client.
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
	Component        EventTarget `json:"component"`
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
	Source           EventTarget `json:"source"`
}

// TouchEvent contains the touch payload sent by the browser client.
type TouchEvent struct {
	ChangedTouches []Touch     `json:"changedTouches"`
	Component      EventTarget `json:"component"`
	Source         EventTarget `json:"source"`
	TargetTouches  []Touch     `json:"targetTouches"`
	Touches        []Touch     `json:"touches"`
	LayerX         int         `json:"layerX"`
	LayerY         int         `json:"layerY"`
	PageX          int         `json:"pageX"`
	PageY          int         `json:"pageY"`
}

// Touch contains one browser touch point from a TouchEvent.
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
	Source        EventTarget `json:"source"`
}

// DragEvent contains the drag payload sent by the browser client.
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
	Component        EventTarget `json:"component"`
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
	Source           EventTarget `json:"source"`
}

// MouseEvent contains the mouse payload sent by the browser client.
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
	Component        EventTarget `json:"component"`
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
	Source           EventTarget `json:"source"`
}

// KeyboardEvent contains the keyboard payload sent by the browser client.
type KeyboardEvent struct {
	IsTrusted        bool        `json:"isTrusted"`
	AltKey           bool        `json:"altKey"`
	Bubbles          bool        `json:"bubbles"`
	Cancelable       bool        `json:"cancelable"`
	Code             string      `json:"code"`
	Composed         bool        `json:"composed"`
	CtrlKey          bool        `json:"ctrlKey"`
	Component        EventTarget `json:"component"`
	DefaultPrevented bool        `json:"defaultPrevented"`
	Detail           int         `json:"detail"`
	EventPhase       int         `json:"eventPhase"`
	IsComposing      bool        `json:"isComposing"`
	Key              string      `json:"key"`
	Location         int         `json:"location"`
	MetaKey          bool        `json:"metaKey"`
	Repeat           bool        `json:"repeat"`
	ShiftKey         bool        `json:"shiftKey"`
	Source           EventTarget `json:"source"`
}

// FormDataEvent contains the form payload sent by the browser client.
type FormDataEvent struct {
	IsTrusted        bool           `json:"isTrusted"`
	Bubbles          bool           `json:"bubbles"`
	Cancelable       bool           `json:"cancelable"`
	Composed         bool           `json:"composed"`
	Component        EventTarget    `json:"component"`
	DefaultPrevented bool           `json:"defaultPrevented"`
	EventPhase       int            `json:"eventPhase"`
	FormData         map[string]any `json:"formData"`
}
