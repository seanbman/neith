package fcmp

import "encoding/json"

// functionName selects the browser/client operation represented by a dispatch.
type functionName string

const (
	auth     functionName = "auth"
	ping     functionName = "ping"
	render   functionName = "render"
	class    functionName = "class"
	dom      functionName = "dom"
	redirect functionName = "redirect"
	event    functionName = "event"
	custom   functionName = "custom"
	fnError  functionName = "error"
)

func newDispatch(key string) *Dispatch {
	return &Dispatch{
		Key: key,
	}
}

// Dispatch is the websocket message exchanged by Go and the browser client.
//
// The flat JSON shape mirrors static/assets/fcmp_types.ts. Function selects
// which nested payload is active for a given message.
type Dispatch struct {
	buf        []byte        `json:"-"`
	conn       *conn         `json:"-"`
	ID         string        `json:"id"`
	Key        string        `json:"key"`
	ConnID     string        `json:"conn_id"`
	HandlerID  string        `json:"handler_id"`
	Action     string        `json:"action"`
	Label      string        `json:"label"`
	Function   functionName  `json:"function"`
	FnEvent    EventListener `json:"event"`
	FnPing     FnPing        `json:"ping"`
	FnRender   FnRender      `json:"render"`
	FnClass    FnClass       `json:"class"`
	FnDOM      FnDOM         `json:"dom"`
	FnRedirect FnRedirect    `json:"redirect"`
	FnCustom   FnCustom      `json:"custom"`
	FnError    FnError       `json:"error"`
}

// listenerStrings serializes listener metadata for the rendered wrapper's
// events attribute.
func (f *FnRender) listenerStrings() string {
	b, err := json.Marshal(f.EventListeners)
	if err != nil {
		return ""
	}
	return string(b)
}
