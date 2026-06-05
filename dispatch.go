package fcmp

import "encoding/json"

// functionName is used  to determine the type of function to run on the client.
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
	_error   functionName = "error"
)

type (
	// FnRender is used internally to render HTML to the client.
	FnRender struct {
		TargetID       string          `json:"target_id"`
		Tag            string          `json:"tag"`
		Inner          bool            `json:"inner"`
		Outer          bool            `json:"outer"`
		Append         bool            `json:"append"`
		Prepend        bool            `json:"prepend"`
		Remove         bool            `json:"remove"`
		HTML           string          `json:"html"`
		EventListeners []EventListener `json:"event_listeners"`
	}
	// FnPing is used internally to ping the client or server.
	FnPing struct {
		Server bool `json:"server"`
		Client bool `json:"client"`
	}
	// FnClass is used internally to add or remove classes from elements.
	FnClass struct {
		TargetID string   `json:"target_id"`
		Remove   bool     `json:"remove"`
		Names    []string `json:"names"`
	}
	// FnDOM is used internally for focused DOM mutations and effects.
	FnDOM struct {
		TargetID  string `json:"target_id"`
		Operation string `json:"operation"`
		Name      string `json:"name,omitempty"`
		Value     string `json:"value,omitempty"`
	}
	// FnRedirect is used internally to redirect the client to a new URL.
	FnRedirect struct {
		URL string `json:"url"`
	}
	// FnCustom is used internally to run custom JavaScript on the client.
	FnCustom struct {
		Function string `json:"function"`
		Data     any    `json:"data"`
		Result   any    `json:"result"`
	}
	// FnError is used internally to log an error on the server if config is set to log errors
	//
	// See: https://pkg.go.dev/github.com/kitkitchen/fcmp#SetConfig
	FnError struct {
		Message string `json:"message"`
	}
)

func newDispatch(key string) *Dispatch {
	return &Dispatch{
		Key: key,
	}
}

// Dispatch contains necessary data for the web api.
//
// While this struct is exported, it is not intended to be used directly and is not exposed during runtime.
//
// See: https://kitkitchen.github.io/docs/fcmp/tutorial/context to read about how Dispatch is used.
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

func (f *FnRender) listenerStrings() string {
	b, err := json.Marshal(f.EventListeners)
	if err != nil {
		return ""
	}
	return string(b)
}
