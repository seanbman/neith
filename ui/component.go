// Package ui provides small renderer-agnostic components for Neith apps.
package ui

import (
	"context"
	"html"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/seanbman/neith"
)

// Option customizes a UI component.
type Option func(*Config)

// Config is the public customization surface passed to Option functions.
type Config struct {
	element *element
}

type element struct {
	tag       string
	attrs     map[string]string
	boolAttrs map[string]bool
	classes   []string
	children  []neith.Component
	text      string
	label     string
	choices   []Choice
	void      bool
}

// Choice describes one select option.
type Choice struct {
	Value    string
	Label    string
	Selected bool
	Disabled bool
}

// Element creates a generic HTML element component.
func Element(tag string, items ...any) neith.Component {
	e := newElement(tag)
	e.apply(items...)
	return e
}

// Fragment renders child components without a wrapper element.
func Fragment(children ...neith.Component) neith.Component {
	return fragment(children)
}

// Text renders escaped text.
func Text(value string) neith.Component {
	return text(value)
}

// Panel groups related content in a section.
func Panel(items ...any) neith.Component {
	return semanticElement("section", "n-panel", items...)
}

// Stack lays out child components as a vertical group.
func Stack(items ...any) neith.Component {
	return semanticElement("div", "n-stack", items...)
}

// Row lays out child components as a horizontal group.
func Row(items ...any) neith.Component {
	return semanticElement("div", "n-row", items...)
}

// Grid lays out child components as a grid group.
func Grid(items ...any) neith.Component {
	return semanticElement("div", "n-grid", items...)
}

// Heading renders an h1 by default. Use Level to choose h2-h6.
func Heading(value string, opts ...Option) neith.Component {
	e := newElement("h1")
	e.text = value
	e.apply(optionsAsItems(opts)...)
	return e
}

// Form renders a form around child components.
func Form(items ...any) neith.Component {
	e := newElement("form")
	e.apply(items...)
	return e
}

// Button renders a button with escaped text.
func Button(label string, opts ...Option) neith.Component {
	e := newElement("button")
	e.text = label
	e.apply(optionsAsItems(opts)...)
	return e
}

// TextInput renders an input, wrapped in a label when Label is supplied.
func TextInput(name string, opts ...Option) neith.Component {
	e := newElement("input")
	e.void = true
	e.attrs["type"] = "text"
	e.attrs["name"] = name
	e.apply(optionsAsItems(opts)...)
	return e
}

// Select renders a select with Choice values supplied through Options.
func Select(name string, opts ...Option) neith.Component {
	e := newElement("select")
	e.attrs["name"] = name
	e.apply(optionsAsItems(opts)...)
	return e
}

// Alert renders a status message.
func Alert(message string, opts ...Option) neith.Component {
	e := newElement("div")
	e.attrs["role"] = "status"
	e.classes = append(e.classes, "n-alert")
	e.text = message
	e.apply(optionsAsItems(opts)...)
	return e
}

// Children appends child components.
func Children(children ...neith.Component) Option {
	return func(c *Config) {
		c.Children(children...)
	}
}

// ID sets the id attribute.
func ID(value string) Option {
	return Attr("id", value)
}

// Name sets the name attribute.
func Name(value string) Option {
	return Attr("name", value)
}

// Type sets the type attribute.
func Type(value string) Option {
	return Attr("type", value)
}

// Value sets the value attribute.
func Value(value string) Option {
	return Attr("value", value)
}

// Placeholder sets the placeholder attribute.
func Placeholder(value string) Option {
	return Attr("placeholder", value)
}

// Method sets the form method attribute.
func Method(value string) Option {
	return Attr("method", value)
}

// Action sets the form action attribute.
func Action(value string) Option {
	return Attr("action", value)
}

// Attr sets an HTML attribute.
func Attr(name string, value string) Option {
	return func(c *Config) {
		c.Attr(name, value)
	}
}

// Class appends one or more class names.
func Class(names ...string) Option {
	return func(c *Config) {
		c.Class(names...)
	}
}

// Label wraps form controls with a visible text label.
func Label(value string) Option {
	return func(c *Config) {
		c.Label(value)
	}
}

// Level changes a Heading to h1-h6.
func Level(level int) Option {
	return func(c *Config) {
		if c == nil || c.element == nil {
			return
		}
		if level < 1 || level > 6 {
			return
		}
		c.element.tag = "h" + strconv.Itoa(level)
	}
}

// Disabled toggles the disabled boolean attribute.
func Disabled(disabled bool) Option {
	return BoolAttr("disabled", disabled)
}

// Required toggles the required boolean attribute.
func Required(required bool) Option {
	return BoolAttr("required", required)
}

// BoolAttr toggles a boolean HTML attribute.
func BoolAttr(name string, enabled bool) Option {
	return func(c *Config) {
		c.BoolAttr(name, enabled)
	}
}

// Options appends select choices with identical labels and values.
func Options(values ...string) Option {
	return func(c *Config) {
		for _, value := range values {
			c.Choices(Choice{Value: value, Label: value})
		}
	}
}

// Choices appends explicit select choices.
func Choices(choices ...Choice) Option {
	return func(c *Config) {
		c.Choices(choices...)
	}
}

// OptionChoice creates one select choice.
func OptionChoice(value string, label string) Choice {
	return Choice{Value: value, Label: label}
}

func semanticElement(tag string, class string, items ...any) neith.Component {
	e := newElement(tag)
	e.classes = append(e.classes, class)
	e.apply(items...)
	return e
}

func newElement(tag string) *element {
	return &element{
		tag:       tag,
		attrs:     map[string]string{},
		boolAttrs: map[string]bool{},
	}
}

func optionsAsItems(opts []Option) []any {
	items := make([]any, 0, len(opts))
	for _, opt := range opts {
		items = append(items, opt)
	}
	return items
}

func (e *element) apply(items ...any) {
	for _, item := range items {
		switch v := item.(type) {
		case nil:
			continue
		case Option:
			v(&Config{element: e})
		case neith.Component:
			e.children = append(e.children, v)
		case string:
			e.children = append(e.children, Text(v))
		case []neith.Component:
			e.children = append(e.children, v...)
		}
	}
}

// Attr sets an HTML attribute.
func (c *Config) Attr(name string, value string) {
	if c == nil || c.element == nil || name == "" {
		return
	}
	c.element.attrs[name] = value
}

// Class appends one or more class names.
func (c *Config) Class(names ...string) {
	if c == nil || c.element == nil {
		return
	}
	for _, name := range names {
		if name != "" {
			c.element.classes = append(c.element.classes, name)
		}
	}
}

// Label sets the visible label used by form controls.
func (c *Config) Label(value string) {
	if c == nil || c.element == nil {
		return
	}
	c.element.label = value
}

// BoolAttr toggles a boolean HTML attribute.
func (c *Config) BoolAttr(name string, enabled bool) {
	if c == nil || c.element == nil || name == "" {
		return
	}
	c.element.boolAttrs[name] = enabled
}

// Children appends child components.
func (c *Config) Children(children ...neith.Component) {
	if c == nil || c.element == nil {
		return
	}
	c.element.children = append(c.element.children, children...)
}

// Choices appends select choices.
func (c *Config) Choices(choices ...Choice) {
	if c == nil || c.element == nil {
		return
	}
	c.element.choices = append(c.element.choices, choices...)
}

func (e *element) Render(ctx context.Context, w io.Writer) error {
	if e.label != "" && isLabelable(e.tag) {
		if _, err := io.WriteString(w, "<label>"); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "<span>"+html.EscapeString(e.label)+"</span>"); err != nil {
			return err
		}
		if err := e.renderElement(ctx, w); err != nil {
			return err
		}
		_, err := io.WriteString(w, "</label>")
		return err
	}
	return e.renderElement(ctx, w)
}

func (e *element) renderElement(ctx context.Context, w io.Writer) error {
	if _, err := io.WriteString(w, "<"+e.tag+e.attrString()+">"); err != nil {
		return err
	}
	if e.void {
		return nil
	}
	if e.text != "" {
		if _, err := io.WriteString(w, html.EscapeString(e.text)); err != nil {
			return err
		}
	}
	for _, choice := range e.choices {
		if err := renderChoice(w, choice); err != nil {
			return err
		}
	}
	for _, child := range e.children {
		if err := child.Render(ctx, w); err != nil {
			return err
		}
	}
	_, err := io.WriteString(w, "</"+e.tag+">")
	return err
}

func (e *element) attrString() string {
	attrs := make(map[string]string, len(e.attrs)+1)
	for key, value := range e.attrs {
		attrs[key] = value
	}
	if len(e.classes) > 0 {
		classes := strings.Join(e.classes, " ")
		if existing, ok := attrs["class"]; ok && existing != "" {
			classes = existing + " " + classes
		}
		attrs["class"] = classes
	}

	keys := make([]string, 0, len(attrs)+len(e.boolAttrs))
	for key := range attrs {
		keys = append(keys, key)
	}
	for key, enabled := range e.boolAttrs {
		_, exists := attrs[key]
		if enabled && !exists {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		if enabled, ok := e.boolAttrs[key]; ok && enabled {
			b.WriteString(" ")
			b.WriteString(html.EscapeString(key))
			continue
		}
		b.WriteString(" ")
		b.WriteString(html.EscapeString(key))
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(attrs[key]))
		b.WriteString(`"`)
	}
	return b.String()
}

func renderChoice(w io.Writer, choice Choice) error {
	label := choice.Label
	if label == "" {
		label = choice.Value
	}
	attrs := []string{`value="` + html.EscapeString(choice.Value) + `"`}
	if choice.Selected {
		attrs = append(attrs, "selected")
	}
	if choice.Disabled {
		attrs = append(attrs, "disabled")
	}
	_, err := io.WriteString(w, "<option "+strings.Join(attrs, " ")+">"+html.EscapeString(label)+"</option>")
	return err
}

func isLabelable(tag string) bool {
	switch tag {
	case "button", "input", "meter", "output", "progress", "select", "textarea":
		return true
	default:
		return false
	}
}

type fragment []neith.Component

func (f fragment) Render(ctx context.Context, w io.Writer) error {
	for _, child := range f {
		if err := child.Render(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

type text string

func (t text) Render(ctx context.Context, w io.Writer) error {
	_, err := io.WriteString(w, html.EscapeString(string(t)))
	return err
}
