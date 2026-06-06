package neith

import "context"

// View creates an interactive Neith component from any renderable component.
//
// It is a readability-focused wrapper around NewFn. The underlying component
// contract stays the same, so templ components, HTML, and custom renderers can
// be migrated into Neith without changing how they render.
func View(ctx context.Context, c Component, opts ...ViewOption) FnComponent {
	f := NewFn(ctx, c)
	for _, opt := range opts {
		if opt != nil {
			f = opt(f)
		}
	}
	return f
}

// ViewOption customizes a View without hiding the underlying FnComponent.
type ViewOption func(FnComponent) FnComponent

// On attaches one handler to one or more browser events.
func On(event OnEvent, handler HandleFn, events ...OnEvent) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.WithEvents(handler, append([]OnEvent{event}, events...)...)
	}
}

// OnSubmit attaches a submit event handler.
func OnSubmit(handler HandleFn) ViewOption {
	return On(EventSubmit, handler)
}

// OnClick attaches a click event handler.
func OnClick(handler HandleFn) ViewOption {
	return On(EventClick, handler)
}

// OnChange attaches a change event handler.
func OnChange(handler HandleFn) ViewOption {
	return On(EventChange, handler)
}

// OnInput attaches an input event handler.
func OnInput(handler HandleFn) ViewOption {
	return On(EventInput, handler)
}

// OnKeyDown attaches a keydown event handler.
func OnKeyDown(handler HandleFn) ViewOption {
	return On(EventKeyDown, handler)
}

// Submit attaches a submit event handler.
func Submit(handler HandleFn) ViewOption {
	return OnSubmit(handler)
}

// Click attaches a click event handler.
func Click(handler HandleFn) ViewOption {
	return OnClick(handler)
}

// Change attaches a change event handler.
func Change(handler HandleFn) ViewOption {
	return OnChange(handler)
}

// Input attaches an input event handler.
func Input(handler HandleFn) ViewOption {
	return OnInput(handler)
}

// KeyDown attaches a keydown event handler.
func KeyDown(handler HandleFn) ViewOption {
	return OnKeyDown(handler)
}

// Label assigns a human-readable debug label to the rendered wrapper.
func Label(label string) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.WithLabel(label)
	}
}

// IntoTag swaps the inner HTML of the first matching tag.
func IntoTag(tag string) ViewOption {
	return SwapTagInner(tag)
}

// IntoElement swaps the inner HTML of one element by ID.
func IntoElement(id string) ViewOption {
	return SwapElementInner(id)
}

// AppendToTag appends the rendered component to the first matching tag.
func AppendToTag(tag string) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.AppendTag(tag)
	}
}

// PrependToTag prepends the rendered component to the first matching tag.
func PrependToTag(tag string) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.PrependTag(tag)
	}
}

// SwapTagInner swaps the inner HTML of the first matching tag.
func SwapTagInner(tag string) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.SwapTagInner(tag)
	}
}

// SwapTagOuter swaps the outer HTML of the first matching tag.
func SwapTagOuter(tag string) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.SwapTagOuter(tag)
	}
}

// AppendToElement appends the rendered component to one element by ID.
func AppendToElement(id string) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.AppendElement(id)
	}
}

// PrependToElement prepends the rendered component to one element by ID.
func PrependToElement(id string) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.PrependElement(id)
	}
}

// SwapElementInner swaps the inner HTML of one element by ID.
func SwapElementInner(id string) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.SwapElementInner(id)
	}
}

// SwapElementOuter swaps the outer HTML of one element by ID.
func SwapElementOuter(id string) ViewOption {
	return func(f FnComponent) FnComponent {
		return f.SwapElementOuter(id)
	}
}
