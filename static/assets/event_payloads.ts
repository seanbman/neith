import type { EventTargetData, Upload } from "./fcmp_types";
import { formValues, uploadFormFiles } from "./uploads";

export type ParsedEventData = {
    data: Object | null;
    uploads: Upload[];
    submitter?: ReturnType<typeof parseEventTarget>;
};

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

function isFormElement(target: EventTarget | null): target is HTMLFormElement {
    return !!target &&
        "tagName" in target &&
        (target as HTMLFormElement).tagName === "FORM";
}

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

function withoutUploads(data: Object | null): ParsedEventData {
    return {
        data,
        uploads: [],
    };
}

function getSubmitterElement(ev: Event) {
    return (ev as SubmitEvent).submitter || null;
}

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
    };
}

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
    };
}

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
    };
}

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
    };
}

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
    target: EventTargetData | null;
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
};

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
};
