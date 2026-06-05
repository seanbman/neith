import { Dispatch, DispatchFunctions, FnEventListener, Fun } from "./fcmp_types";

export class API {
    private ws: WebSocket | null = null;

    constructor(ws: WebSocket) {
        this.ws = ws;
    }

    public Process(d: Dispatch) {
        switch (d.function) {
            case Fun.REDIRECT:
                window.location.href = d.redirect.url;
                break;
            default:
                if (!this.funs[d.function]) {
                    this.Error(d, "function not found: " + d.function);
                    break;
                }
                const result = this.funs[d.function](d);
                if (!result) break;
                this.Dispatch(result);
                break;
        }
    }

    private Dispatch = (data: Dispatch | void) => {
        if (!data) return;
        if (!this.ws) {
            throw new Error("ws: not connected to server...");
        }
        this.ws.send(JSON.stringify(data));
    };

    private funs: DispatchFunctions = {
        ping: (d: Dispatch) => {
            d.ping.client = true;
            return d;
        },
        render: (d: Dispatch) => {
            const elem = this.findRenderTarget(d);
            if (!elem) return;

            const html = d.render.html;

            if (d.render.inner) {
                elem.innerHTML = html;
            }
            if (d.render.outer) {
                elem.outerHTML = html;
            }
            if (d.render.append) {
                elem.innerHTML += html;
            }
            if (d.render.prepend) {
                elem.innerHTML = html + elem.innerHTML;
            }
            if (d.render.remove) {
                elem.remove();
                return;
            }

            d = this.utils.parseEventListeners(elem, d);
            this.utils.addEventListeners(d);

            return;
        },
        class: (d: Dispatch) => {
            const elem = document.getElementById(d.class.target_id);
            if (!elem) {
                return this.Error(d, "element not found");
            }
            if (d.class.remove) {
                elem.classList.remove(...d.class.names);
            } else {
                elem.classList.add(...d.class.names);
            }
            return;
        },
        custom: (d: Dispatch) => {
            const fn = (window as unknown as Record<string, (data: Object) => Object | undefined>)[d.custom.function];
            if (typeof fn !== "function") {
                return this.Error(d, "custom function not found: " + d.custom.function);
            }
            d.custom.result = fn(d.custom.data);
            return d;
        },
    };

    private findRenderTarget(d: Dispatch): Element | void {
        if (d.render.tag != "") {
            const elem = document.getElementsByTagName(d.render.tag)[0];
            if (!elem) {
                return this.Error(
                    d,
                    "element with tag not found: " + d.render.tag
                );
            }
            return elem;
        }

        if (d.render.target_id != "") {
            const elem = document.getElementById(d.render.target_id);
            if (!elem) {
                return this.Error(
                    d,
                    "element with target_id not found: " +
                    d.render.target_id
                );
            }
            return elem;
        }

        return this.Error(d, "no target or tag specified");
    }

    private utils = {
        parseEventListeners: (element: Element, d: Dispatch): Dispatch => {
            const events = this.utils.getAttributes(element, "events");
            const listeners = events.map((e) => {
                const event = JSON.parse(e);
                if (!event) return;
                return event as FnEventListener[];
            });
            d.render.event_listeners = listeners.flat().filter((e) => e != null);
            return d;
        },
        // Element selectors
        parseFormData: (ev: Event) => {
            const form = this.utils.getFormFromEvent(ev);
            if (!form) {
                return ParseEventTarget(ev.target);
            }
            const formData = new FormData(form);
            return Object.fromEntries(formData.entries());
        },
        getFormFromEvent: (ev: Event): HTMLFormElement | null => {
            const target = ev.target;
            if (target instanceof HTMLFormElement) {
                return target;
            }
            if (target instanceof Element) {
                return target.closest("form");
            }
            return null;
        },
        getAttributes: (elem: Element, attribute: string): string[] => {
            const elems = elem.querySelectorAll(`[${attribute}]`);
            return Array.from(elems)
                .map((el) => el.getAttribute(attribute))
                .filter((value): value is string => value !== null);
        },
        addEventListeners: (d: Dispatch) => {
            if (!d.render.event_listeners) return;
            // Event listeners
            d.render.event_listeners.forEach((listener: FnEventListener) => {
                let elem = document.getElementById(listener.target_id);
                if (!elem) {
                    this.Error(d, "element not found");
                    return;
                }
                if (elem.firstChild) {
                    elem = elem.firstChild as HTMLElement;
                }
                elem.addEventListener(listener.on, (ev) => {
                    ev.preventDefault();
                    const eventDispatch = cloneDispatch(d);
                    eventDispatch.function = Fun.EVENT;
                    eventDispatch.event = { ...listener };
                    eventDispatch.event.data = this.utils.parseEventData(listener.on, ev);
                    this.Dispatch(eventDispatch);
                });
            });
        },
        parseEventData: (eventName: string, ev: Event) => {
            if (["submit", "change"].includes(eventName)) {
                return this.utils.parseFormData(ev);
            }
            if (["pointerdown", "pointerup", "pointermove", "click", "contextmenu", "dblclick"].includes(eventName)) {
                return ParsePointerEvent(ev as PointerEvent);
            }
            if (["drag", "dragend", "dragenter", "dragexitcapture", "dragleave", "dragover", "dragstart", "drop"].includes(eventName)) {
                return ParseDragEvent(ev as DragEvent);
            }
            if (["mousedown", "mouseup", "mousemove"].includes(eventName)) {
                return ParseMouseEvent(ev as MouseEvent);
            }
            if (["keydown", "keyup", "keypress"].includes(eventName)) {
                return ParseKeyboardEvent(ev as KeyboardEvent);
            }
            if (["touchstart", "touchend", "touchmove", "touchcancel"].includes(eventName)) {
                return ParseTouchEvent(ev as TouchEvent & { layerX: number; layerY: number; pageX: number; pageY: number });
            }
            return ParseEventTarget(ev.target);
        },
    };

    private Error = (d: Dispatch, message: string) => {
        d.function = Fun.ERROR;
        d.error = { message };
        this.Dispatch(d);
    };
}

function cloneDispatch(d: Dispatch): Dispatch {
    return JSON.parse(JSON.stringify(d)) as Dispatch;
}

function ParseEventTarget(ev: any) {
    if (!ev) return null;
    return {
        id: ev.id || "",
        name: ev.name || "",
        tagName: ev.tagName || "",
        innerHTML: ev.innerHTML || "",
        outerHTML: ev.outerHTML || "",
        value: ev.value || "",
    } as Partial<EventTarget>;
}

function ParsePointerEvent(ev: PointerEvent): PointerEventProperties {
    return {
        isTrusted: ev.isTrusted,
        altKey: ev.altKey,
        bubbles: ev.bubbles,
        button: ev.button,
        buttons: ev.buttons,
        cancelable: ev.cancelable,
        clientX: ev.clientX,
        clientY: ev.clientY,
        composed: ev.composed,
        ctrlKey: ev.ctrlKey,
        currentTarget: ParseEventTarget(ev.currentTarget),
        defaultPrevented: ev.defaultPrevented,
        detail: ev.detail,
        eventPhase: ev.eventPhase,
        height: ev.height,
        isPrimary: ev.isPrimary,
        metaKey: ev.metaKey,
        movementX: ev.movementX,
        movementY: ev.movementY,
        offsetX: ev.offsetX,
        offsetY: ev.offsetY,
        pageX: ev.pageX,
        pageY: ev.pageY,
        pointerId: ev.pointerId,
        pointerType: ev.pointerType,
        pressure: ev.pressure,
        relatedTarget: ParseEventTarget(ev.relatedTarget),
    };
}

function ParseTouchEvent(ev: TouchEvent & { layerX: number; layerY: number; pageX: number; pageY: number}): TouchEventProperties & { layerX: number; layerY: number; pageX: number; pageY: number }{
    return {
        changedTouches: Array.from(ev.changedTouches).map((t) => ParseTouch(t)),
        targetTouches: Array.from(ev.targetTouches).map((t) => ParseTouch(t)),
        touches: Array.from(ev.touches).map((t) => ParseTouch(t)),
        layerX: ev.layerX,
        layerY: ev.layerY,
        pageX: ev.pageX,
        pageY: ev.pageY,
    };
}

function ParseTouch(ev: Touch): TouchProperties {
    return {
        clientX: ev.clientX,
        clientY: ev.clientY,
        identifier: ev.identifier,
        pageX: ev.pageX,
        pageY: ev.pageY,
        radiusX: ev.radiusX,
        radiusY: ev.radiusY,
        rotationAngle: ev.rotationAngle,
        screenX: ev.screenX,
        screenY: ev.screenY,
        target: ParseEventTarget(ev.target),
    };
}

function ParseDragEvent(ev: DragEvent): DragEventProperties {
    return {
        isTrusted: ev.isTrusted,
        altKey: ev.altKey,
        bubbles: ev.bubbles,
        button: ev.button,
        buttons: ev.buttons,
        cancelable: ev.cancelable,
        clientX: ev.clientX,
        clientY: ev.clientY,
        composed: ev.composed,
        ctrlKey: ev.ctrlKey,
        currentTarget: ParseEventTarget(ev.currentTarget),
        defaultPrevented: ev.defaultPrevented,
        detail: ev.detail,
        eventPhase: ev.eventPhase,
        metaKey: ev.metaKey,
        movementX: ev.movementX,
        movementY: ev.movementY,
        offsetX: ev.offsetX,
        offsetY: ev.offsetY,
        pageX: ev.pageX,
        pageY: ev.pageY,
        relatedTarget: ParseEventTarget(ev.relatedTarget),
    };
}

function ParseMouseEvent(ev: MouseEvent): MouseEventProperties {
    return {
        isTrusted: ev.isTrusted,
        altKey: ev.altKey,
        bubbles: ev.bubbles,
        button: ev.button,
        buttons: ev.buttons,
        cancelable: ev.cancelable,
        clientX: ev.clientX,
        clientY: ev.clientY,
        composed: ev.composed,
        ctrlKey: ev.ctrlKey,
        currentTarget: ParseEventTarget(ev.currentTarget),
        defaultPrevented: ev.defaultPrevented,
        detail: ev.detail,
        eventPhase: ev.eventPhase,
        metaKey: ev.metaKey,
        movementX: ev.movementX,
        movementY: ev.movementY,
        offsetX: ev.offsetX,
        offsetY: ev.offsetY,
        pageX: ev.pageX,
        pageY: ev.pageY,
        relatedTarget: ParseEventTarget(ev.relatedTarget),
    };
}

function ParseKeyboardEvent(ev: KeyboardEvent): KeyboardEventProperties {
    return {
        isTrusted: ev.isTrusted,
        altKey: ev.altKey,
        bubbles: ev.bubbles,
        cancelable: ev.cancelable,
        code: ev.code,
        composed: ev.composed,
        ctrlKey: ev.ctrlKey,
        currentTarget: ParseEventTarget(ev.currentTarget),
        defaultPrevented: ev.defaultPrevented,
        detail: ev.detail,
        eventPhase: ev.eventPhase,
        isComposing: ev.isComposing,
        key: ev.key,
        location: ev.location,
        metaKey: ev.metaKey,
        repeat: ev.repeat,
        shiftKey: ev.shiftKey,
    };
}

// Event types
type PointerEventProperties = {
    isTrusted: boolean;
    altKey: boolean;
    bubbles: boolean;
    button: number;
    buttons: number;
    cancelable: boolean;
    clientX: number;
    clientY: number;
    composed: boolean;
    ctrlKey: boolean;
    currentTarget: Partial<EventTarget> | null;
    defaultPrevented: boolean;
    detail: number;
    eventPhase: number;
    height: number;
    isPrimary: boolean;
    metaKey: boolean;
    movementX: number;
    movementY: number;
    offsetX: number;
    offsetY: number;
    pageX: number;
    pageY: number;
    pointerId: number;
    pointerType: string;
    pressure: number;
    relatedTarget: Partial<EventTarget> | null;
};

type TouchEventProperties = {
    changedTouches: TouchProperties[];
    targetTouches: TouchProperties[];
    touches: TouchProperties[];
    layerX: number;
    layerY: number;
    pageX: number;
    pageY: number;
};

type TouchProperties = {
    clientX: number;
    clientY: number;
    identifier: number;
    pageX: number;
    pageY: number;
    radiusX: number;
    radiusY: number;
    rotationAngle: number;
    screenX: number;
    screenY: number;
    target: Partial<EventTarget> | null;
};

type DragEventProperties = {
    isTrusted: boolean;
    altKey: boolean;
    bubbles: boolean;
    button: number;
    buttons: number;
    cancelable: boolean;
    clientX: number;
    clientY: number;
    composed: boolean;
    ctrlKey: boolean;
    currentTarget: Partial<EventTarget> | null;
    defaultPrevented: boolean;
    detail: number;
    eventPhase: number;
    metaKey: boolean;
    movementX: number;
    movementY: number;
    offsetX: number;
    offsetY: number;
    pageX: number;
    pageY: number;
    relatedTarget: Partial<EventTarget> | null;
};

type MouseEventProperties = {
    isTrusted: boolean;
    altKey: boolean;
    bubbles: boolean;
    button: number;
    buttons: number;
    cancelable: boolean;
    clientX: number;
    clientY: number;
    composed: boolean;
    ctrlKey: boolean;
    currentTarget: Partial<EventTarget> | null;
    defaultPrevented: boolean;
    detail: number;
    eventPhase: number;
    metaKey: boolean;
    movementX: number;
    movementY: number;
    offsetX: number;
    offsetY: number;
    pageX: number;
    pageY: number;
    relatedTarget: Partial<EventTarget> | null;
};

type KeyboardEventProperties = {
    isTrusted: boolean;
    altKey: boolean;
    bubbles: boolean;
    cancelable: boolean;
    code: string;
    composed: boolean;
    ctrlKey: boolean;
    currentTarget: Partial<EventTarget> | null;
    defaultPrevented: boolean;
    detail: number;
    eventPhase: number;
    isComposing: boolean;
    key: string;
    location: number;
    metaKey: boolean;
    repeat: boolean;
    shiftKey: boolean;
};
