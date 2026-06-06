package neith

// FnRender describes how rendered HTML should be applied in the browser.
type FnRender struct {
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

// FnPing confirms that the websocket is still alive on both sides.
type FnPing struct {
	Server bool `json:"server"`
	Client bool `json:"client"`
}

// FnClass adds or removes CSS classes from a browser element.
type FnClass struct {
	TargetID string   `json:"target_id"`
	Remove   bool     `json:"remove"`
	Names    []string `json:"names"`
}

// FnDOM applies a focused DOM mutation such as attribute, style, text, value,
// focus, blur, scroll, enable, disable, or removal.
type FnDOM struct {
	TargetID  string `json:"target_id"`
	Operation string `json:"operation"`
	Name      string `json:"name,omitempty"`
	Value     string `json:"value,omitempty"`
}

// FnRedirect sends the browser to a new URL.
type FnRedirect struct {
	URL string `json:"url"`
}

// FnCustom calls a named browser function and receives its result.
type FnCustom struct {
	Function string `json:"function"`
	Data     any    `json:"data"`
	Result   any    `json:"result"`
}

// FnError carries an error message between Go and the browser client.
type FnError struct {
	Message string `json:"message"`
}
