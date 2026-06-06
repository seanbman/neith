import type { EventTargetData, Upload } from "./fcmp_types";
import { formValues, uploadFormFiles } from "./uploads";

/**
 * Normalized result returned by every event parser.
 *
 * `data` is the JSON-safe event payload sent to Go through EventData.
 * `uploads` contains file metadata produced by the HTTP upload endpoint.
 * `submitter` is populated for form submit events when the browser exposes the
 * button or input that submitted the form.
 */
export type ParsedEventData = {
    data: Object | null;
    uploads: Upload[];
    submitter?: ReturnType<typeof parseEventTarget>;
};

/**
 * Converts a browser Event into the fcmp event payload sent to Go.
 *
 * The event name comes from the server-provided listener metadata. It lets the
 * parser choose a smaller, explicit payload shape instead of trying to serialize
 * the native browser Event object, which contains circular references and many
 * non-JSON values.
 */
export function parseEventData(eventName: string, ev: Event) {
    if (["submit", "change"].includes(eventName)) {
        return parseFormData(ev);
    }
    if (["pointerdown", "pointerup", "pointermove", "click", "contextmenu", "dblclick"].includes(eventName)) {
        return withoutUploads(parsePointerEvent(ev as PointerEvent));
    }
    if (["drag", "dragend", "dragenter", "dragexitcapture", "dragleave", "dragover", "dragstart", "drop"].includes(eventName)) {
        return withoutUploads(parseDragEvent(ev as DragEvent));
    }
    if (["mousedown", "mouseup", "mousemove"].includes(eventName)) {
        return withoutUploads(parseMouseEvent(ev as MouseEvent));
    }
    if (["keydown", "keyup", "keypress"].includes(eventName)) {
        return withoutUploads(parseKeyboardEvent(ev as KeyboardEvent));
    }
    if (["touchstart", "touchend", "touchmove", "touchcancel"].includes(eventName)) {
        return withoutUploads(parseTouchEvent(ev as TouchEvent & { layerX: number; layerY: number; pageX: number; pageY: number }));
    }
    return withoutUploads(parseEventTarget(ev.target));
}

/**
 * Builds event data for form-like events.
 *
 * Forms are special because file inputs need an HTTP upload before the websocket
 * event is sent. Normal form fields are still returned as data, while uploaded
 * files are represented by server-provided Upload metadata.
 */
async function parseFormData(ev: Event): Promise<ParsedEventData> {
    const form = getFormFromEvent(ev);
    if (!form) {
        return withoutUploads(parseEventTarget(ev.target));
    }
    const submitter = getSubmitterElement(ev);
    const uploads = await uploadFormFiles(form);
    return {
        data: formValues(form, submitter),
        uploads,
        submitter: parseEventTarget(submitter),
    };
}

/**
 * Finds the form associated with a submit or change event.
 *
 * Submit events usually target the form itself. Change events may target a child
 * input, so this falls back to closest("form") when possible.
 */
function getFormFromEvent(ev: Event): HTMLFormElement | null {
    const target = ev.target;
    if (isFormElement(target)) {
        return target;
    }
    if (target && typeof (target as Element).closest === "function") {
        return (target as Element).closest("form");
    }
    return null;
}

/**
 * Type guard for DOM targets that are real HTMLFormElement instances.
 */
function isFormElement(target: EventTarget | null): target is HTMLFormElement {
    return !!target &&
        "tagName" in target &&
        (target as HTMLFormElement).tagName === "FORM";
}

/**
 * Captures a JSON-safe snapshot of a DOM event target.
 *
 * Native EventTarget objects are not serializable. This snapshot keeps the
 * useful parts a Go handler usually needs: identity, form values, classes,
 * attributes, dataset entries, and selected option values.
 */
export function parseEventTarget(ev: any): EventTargetData | null {
    if (!ev) return null;
    return {
        id: ev.id || "",
        name: ev.name || "",
        classList: ev.classList ? Array.from(ev.classList).map(String) : [],
        tagName: ev.tagName || "",
        innerHTML: ev.innerHTML || "",
        outerHTML: ev.outerHTML || "",
        value: ev.value || "",
        checked: !!ev.checked,
        disabled: !!ev.disabled,
        hidden: !!ev.hidden,
        style: ev.getAttribute ? ev.getAttribute("style") || "" : "",
        attributes: ev.attributes ? Array.from(ev.attributes).map((attr: Attr) => `${attr.name}=${attr.value}`) : [],
        dataset: ev.dataset ? Object.entries(ev.dataset).map(([key, value]) => `${key}=${value || ""}`) : [],
        selectedOptions: ev.selectedOptions ? Array.from(ev.selectedOptions).map((option: HTMLOptionElement) => option.value) : [],
    };
}

/**
 * Wraps non-form event data in the common ParsedEventData shape.
 */
function withoutUploads(data: Object | null): ParsedEventData {
    return {
        data,
        uploads: [],
    };
}

/**
 * Returns the button or input that submitted a form, when the browser exposes it.
 */
function getSubmitterElement(ev: Event) {
    return (ev as SubmitEvent).submitter || null;
}

/**
 * Extracts the fields fcmp preserves from PointerEvent.
 */
function parsePointerEvent(ev: PointerEvent): PointerEventProperties {
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
        currentTarget: parseEventTarget(ev.currentTarget),
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
        relatedTarget: parseEventTarget(ev.relatedTarget),
        target: parseEventTarget(ev.target),
    };
}

/**
 * Extracts touch lists and coordinates from TouchEvent.
 *
 * TouchEvent objects contain TouchList collections. Converting them to arrays
 * keeps the payload JSON-safe and easy for Go to unmarshal.
 */
function parseTouchEvent(ev: TouchEvent & { layerX: number; layerY: number; pageX: number; pageY: number }): TouchEventProperties & { layerX: number; layerY: number; pageX: number; pageY: number } {
    return {
        changedTouches: Array.from(ev.changedTouches).map((t) => parseTouch(t)),
        targetTouches: Array.from(ev.targetTouches).map((t) => parseTouch(t)),
        touches: Array.from(ev.touches).map((t) => parseTouch(t)),
        layerX: ev.layerX,
        layerY: ev.layerY,
        pageX: ev.pageX,
        pageY: ev.pageY,
    };
}

/**
 * Extracts one serializable touch point from a browser Touch object.
 */
function parseTouch(ev: Touch): TouchProperties {
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
        target: parseEventTarget(ev.target),
    };
}

/**
 * Extracts the fields fcmp preserves from DragEvent.
 */
function parseDragEvent(ev: DragEvent): DragEventProperties {
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
        currentTarget: parseEventTarget(ev.currentTarget),
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
        relatedTarget: parseEventTarget(ev.relatedTarget),
        target: parseEventTarget(ev.target),
    };
}

/**
 * Extracts the fields fcmp preserves from MouseEvent.
 */
function parseMouseEvent(ev: MouseEvent): MouseEventProperties {
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
        currentTarget: parseEventTarget(ev.currentTarget),
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
        relatedTarget: parseEventTarget(ev.relatedTarget),
        target: parseEventTarget(ev.target),
    };
}

/**
 * Extracts the fields fcmp preserves from KeyboardEvent.
 */
function parseKeyboardEvent(ev: KeyboardEvent): KeyboardEventProperties {
    return {
        isTrusted: ev.isTrusted,
        altKey: ev.altKey,
        bubbles: ev.bubbles,
        cancelable: ev.cancelable,
        code: ev.code,
        composed: ev.composed,
        ctrlKey: ev.ctrlKey,
        currentTarget: parseEventTarget(ev.currentTarget),
        defaultPrevented: ev.defaultPrevented,
        detail: ev.detail,
        eventPhase: ev.eventPhase,
        isComposing: ev.isComposing,
        key: ev.key,
        location: ev.location,
        metaKey: ev.metaKey,
        repeat: ev.repeat,
        shiftKey: ev.shiftKey,
        target: parseEventTarget(ev.target),
    };
}

/**
 * JSON shape used by Go's PointerEvent struct.
 */
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
    currentTarget: EventTargetData | null;
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
    relatedTarget: EventTargetData | null;
    target: EventTargetData | null;
};

/**
 * JSON shape used by Go's TouchEvent struct.
 */
type TouchEventProperties = {
    changedTouches: TouchProperties[];
    targetTouches: TouchProperties[];
    touches: TouchProperties[];
    layerX: number;
    layerY: number;
    pageX: number;
    pageY: number;
};

/**
 * JSON shape used by Go's Touch struct.
 */
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
    target: EventTargetData | null;
};

/**
 * JSON shape used by Go's DragEvent struct.
 */
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
    currentTarget: EventTargetData | null;
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
    relatedTarget: EventTargetData | null;
    target: EventTargetData | null;
};

/**
 * JSON shape used by Go's MouseEvent struct.
 */
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
    currentTarget: EventTargetData | null;
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
    relatedTarget: EventTargetData | null;
    target: EventTargetData | null;
};

/**
 * JSON shape used by Go's KeyboardEvent struct.
 */
type KeyboardEventProperties = {
    isTrusted: boolean;
    altKey: boolean;
    bubbles: boolean;
    cancelable: boolean;
    code: string;
    composed: boolean;
    ctrlKey: boolean;
    currentTarget: EventTargetData | null;
    defaultPrevented: boolean;
    detail: number;
    eventPhase: number;
    isComposing: boolean;
    key: string;
    location: number;
    metaKey: boolean;
    repeat: boolean;
    shiftKey: boolean;
    target: EventTargetData | null;
};
