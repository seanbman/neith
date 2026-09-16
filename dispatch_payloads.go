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

// Mutation is one allowlisted browser-side change. Multiple mutations may be
// carried in one mutate instruction so related UI changes stay ordered.
type Mutation struct {
	Operation string   `json:"operation"`
	Names     []string `json:"names,omitempty"`
	Name      string   `json:"name,omitempty"`
	Value     string   `json:"value,omitempty"`
}

// FnMutate applies a bounded batch of allowlisted changes to one element.
type FnMutate struct {
	TargetID  string     `json:"target_id"`
	Mutations []Mutation `json:"mutations"`
}

// FnClass is the transitional pre-Phase-3 class payload. New server helpers
// translate class changes to FnMutate rather than emitting this wire operation.
type FnClass struct {
	TargetID string   `json:"target_id"`
	Remove   bool     `json:"remove"`
	Names    []string `json:"names"`
}

// FnDOM is the transitional pre-Phase-3 DOM payload. New server helpers
// translate focused DOM changes to FnMutate rather than emitting this operation.
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
