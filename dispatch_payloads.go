package neith

// FnRender describes how rendered HTML should be applied in the browser.
type FnRender struct {
	TargetID string `json:"target_id"`; Tag string `json:"tag"`; Inner bool `json:"inner"`; Outer bool `json:"outer"`; Append bool `json:"append"`; Prepend bool `json:"prepend"`; Remove bool `json:"remove"`; HTML string `json:"html"`; EventListeners []EventListener `json:"event_listeners"`
}
type FnPing struct { Server bool `json:"server"`; Client bool `json:"client"` }
type Mutation struct { Operation string `json:"operation"`; Names []string `json:"names,omitempty"`; Name string `json:"name,omitempty"`; Value string `json:"value,omitempty"` }
type FnMutate struct { TargetID string `json:"target_id"`; Mutations []Mutation `json:"mutations"` }
// Transitional v1 compatibility payloads.
type FnClass struct { TargetID string `json:"target_id"`; Remove bool `json:"remove"`; Names []string `json:"names"` }
type FnDOM struct { TargetID string `json:"target_id"`; Operation string `json:"operation"`; Name string `json:"name,omitempty"`; Value string `json:"value,omitempty"` }
// FnNavigate is the Phase 3 finite browser navigation primitive.
type FnNavigate struct { URL string `json:"url"` }
// FnRedirect remains readable for v1 compatibility; new helpers emit navigate.
type FnRedirect struct { URL string `json:"url"` }
type FnCustom struct { Function string `json:"function"`; Data any `json:"data"`; Result any `json:"result"` }
type FnError struct { Message string `json:"message"` }
