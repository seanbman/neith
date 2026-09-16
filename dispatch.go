package neith

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// ProtocolVersion is the current Neith server/browser wire protocol version.
const ProtocolVersion = 1

type functionName string

const (
	ping     functionName = "ping"
	render   functionName = "render"
	mutate   functionName = "mutate"
	class    functionName = "class" // transitional v1 compatibility
	dom      functionName = "dom"   // transitional v1 compatibility
	redirect functionName = "redirect"
	event    functionName = "event"
	custom   functionName = "custom"
	fnError  functionName = "error"
)

func newDispatch(key string) *Dispatch {
	return &Dispatch{Version: ProtocolVersion, Key: key, rt: defaultRuntime}
}

// Dispatch is Neith's internal representation of one protocol operation.
type Dispatch struct {
	buf        []byte       `json:"-"`
	conn       *conn        `json:"-"`
	rt         *runtime     `json:"-"`
	Version    int          `json:"-"`
	ID         string       `json:"-"`
	Key        string       `json:"-"`
	ConnID     string       `json:"-"`
	HandlerID  string       `json:"-"`
	Action     string       `json:"-"`
	Label      string       `json:"-"`
	Function   functionName `json:"-"`
	FnEvent    EventListener
	FnPing     FnPing
	FnRender   FnRender
	FnMutate   FnMutate
	FnClass    FnClass
	FnDOM      FnDOM
	FnRedirect FnRedirect
	FnCustom   FnCustom
	FnError    FnError
}

type protocolEnvelope struct {
	Version   int             `json:"v"`
	Type      functionName    `json:"type"`
	ID        string          `json:"id,omitempty"`
	Key       string          `json:"key,omitempty"`
	ConnID    string          `json:"conn_id,omitempty"`
	HandlerID string          `json:"handler_id,omitempty"`
	Action    string          `json:"action,omitempty"`
	Label     string          `json:"label,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

func (d Dispatch) MarshalJSON() ([]byte, error) {
	version := d.Version
	if version == 0 { version = ProtocolVersion }
	if !knownFunction(d.Function) { return nil, fmt.Errorf("neith: unknown protocol type %q", d.Function) }
	payload, err := json.Marshal(d.payload())
	if err != nil { return nil, fmt.Errorf("neith: encode %s payload: %w", d.Function, err) }
	return json.Marshal(protocolEnvelope{Version: version, Type: d.Function, ID: d.ID, Key: d.Key, ConnID: d.ConnID, HandlerID: d.HandlerID, Action: d.Action, Label: d.Label, Payload: payload})
}

func (d *Dispatch) UnmarshalJSON(data []byte) error {
	var envelope protocolEnvelope
	if err := decodeJSONStrict(data, &envelope); err != nil { return fmt.Errorf("neith: invalid protocol envelope: %w", err) }
	if envelope.Version != ProtocolVersion { return fmt.Errorf("neith: unsupported protocol version %d", envelope.Version) }
	if !knownFunction(envelope.Type) { return fmt.Errorf("neith: unknown protocol type %q", envelope.Type) }
	if len(envelope.Payload) == 0 || bytes.Equal(envelope.Payload, []byte("null")) { return errors.New("neith: protocol payload is required") }
	*d = Dispatch{Version: envelope.Version, Function: envelope.Type, ID: envelope.ID, Key: envelope.Key, ConnID: envelope.ConnID, HandlerID: envelope.HandlerID, Action: envelope.Action, Label: envelope.Label}
	if err := d.decodePayload(envelope.Payload); err != nil { return fmt.Errorf("neith: invalid %s payload: %w", envelope.Type, err) }
	return nil
}

func (d Dispatch) payload() any {
	switch d.Function {
	case ping: return d.FnPing
	case render: return d.FnRender
	case mutate: return d.FnMutate
	case class: return d.FnClass
	case dom: return d.FnDOM
	case redirect: return d.FnRedirect
	case event: return d.FnEvent
	case custom: return d.FnCustom
	case fnError: return d.FnError
	default: return nil
	}
}

func (d *Dispatch) decodePayload(raw json.RawMessage) error {
	switch d.Function {
	case ping: return decodeJSONStrict(raw, &d.FnPing)
	case render: return decodeJSONStrict(raw, &d.FnRender)
	case mutate: return decodeJSONStrict(raw, &d.FnMutate)
	case class: return decodeJSONStrict(raw, &d.FnClass)
	case dom: return decodeJSONStrict(raw, &d.FnDOM)
	case redirect: return decodeJSONStrict(raw, &d.FnRedirect)
	case event: return decodeJSONStrict(raw, &d.FnEvent)
	case custom: return decodeJSONStrict(raw, &d.FnCustom)
	case fnError: return decodeJSONStrict(raw, &d.FnError)
	default: return fmt.Errorf("unknown protocol type %q", d.Function)
	}
}

func decodeJSONStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil { return err }
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil { return errors.New("multiple JSON values") }
		return err
	}
	return nil
}

func knownFunction(fn functionName) bool {
	switch fn {
	case ping, render, mutate, class, dom, redirect, event, custom, fnError: return true
	default: return false
	}
}

func (d Dispatch) validVersion() bool { return d.Version == ProtocolVersion }

func (d Dispatch) validInboundFunction() bool {
	switch d.Function {
	case ping, event, custom, fnError: return true
	default: return false
	}
}

func (f *FnRender) listenerStrings() string {
	b, err := json.Marshal(f.EventListeners)
	if err != nil { return "" }
	return string(b)
}
