import type { ErrorReporter } from "./events";
import type { Dispatch } from "./neith_types";

/**
 * Applies an HTML render dispatch to the requested DOM target.
 *
 * Go decides whether the incoming HTML should replace, append, prepend, remove,
 * or swap a DOM node. This function only performs that DOM operation and returns
 * the element that was targeted so the caller can scan it for neith event
 * metadata after the HTML is in place.
 */
export function applyRender(d: Dispatch, reportError: ErrorReporter): Element | void {
    const elem = findRenderTarget(d, reportError);
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

    return elem;
}

/**
 * Adds or removes one or more classes from an element by ID.
 *
 * Class updates are intentionally narrower than full renders. They let Go
 * update visual state without replacing HTML or disturbing event listeners.
 */
export function applyClass(d: Dispatch, reportError: ErrorReporter) {
    const elem = document.getElementById(d.class.target_id);
    if (!elem) {
        return reportError(d, "element not found");
    }
    if (d.class.remove) {
        elem.classList.remove(...d.class.names);
    } else {
        elem.classList.add(...d.class.names);
    }
    return;
}

/**
 * Applies a small DOM operation sent by the server.
 *
 * These operations cover common imperative updates, such as changing an
 * attribute, setting text, focusing an input, or disabling a button. The server
 * sends a string operation name so the protocol can remain simple JSON.
 */
export function applyDOM(d: Dispatch, reportError: ErrorReporter) {
    const elem = document.getElementById(d.dom.target_id);
    if (!elem) {
        return reportError(d, "element not found");
    }

    switch (d.dom.operation) {
        case "setAttribute":
            elem.setAttribute(d.dom.name, d.dom.value);
            return;
        case "removeAttribute":
            elem.removeAttribute(d.dom.name);
            return;
        case "setStyle":
            elem.style.setProperty(d.dom.name, d.dom.value);
            return;
        case "removeStyle":
            elem.style.removeProperty(d.dom.name);
            return;
        case "setText":
            elem.textContent = d.dom.value;
            return;
        case "setValue":
            setElementValue(elem, d.dom.value);
            return;
        case "focus":
            focusElement(elem);
            return;
        case "blur":
            blurElement(elem);
            return;
        case "scrollIntoView":
            scrollElementIntoView(elem);
            return;
        case "disable":
            setElementDisabled(elem, true);
            return;
        case "enable":
            setElementDisabled(elem, false);
            return;
        default:
            return reportError(d, "dom operation not found: " + d.dom.operation);
    }
}

/**
 * Calls a browser function by name and stores the return value on the dispatch.
 *
 * This is the escape hatch for application-specific JavaScript. The named
 * function must exist on window, and its result is sent back to Go in
 * d.custom.result.
 */
export function applyCustom(d: Dispatch, reportError: ErrorReporter): Dispatch | void {
    const fn = (window as unknown as Record<string, (data: Object) => Object | undefined>)[d.custom.function];
    if (typeof fn !== "function") {
        return reportError(d, "custom function not found: " + d.custom.function);
    }
    d.custom.result = fn(d.custom.data);
    return d;
}

/**
 * Resolves the DOM node targeted by a render dispatch.
 *
 * Render dispatches can target the first matching tag, such as `main`, or a
 * specific element ID. If neither target is present, the error reporter sends a
 * protocol-level error back to Go.
 */
function findRenderTarget(d: Dispatch, reportError: ErrorReporter): Element | void {
    if (d.render.tag != "") {
        const elem = document.getElementsByTagName(d.render.tag)[0];
        if (!elem) {
            return reportError(
                d,
                "element with tag not found: " + d.render.tag
            );
        }
        return elem;
    }

    if (d.render.target_id != "") {
        const elem = document.getElementById(d.render.target_id);
        if (!elem) {
            return reportError(
                d,
                "element with target_id not found: " +
                d.render.target_id
            );
        }
        return elem;
    }

    return reportError(d, "no target or tag specified");
}

/**
 * Sets the `.value` property when the target is a value-bearing form control.
 */
function setElementValue(elem: Element, value: string) {
    if ("value" in elem) {
        (elem as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value = value;
    }
}

/**
 * Focuses an element only when the browser exposes a callable focus method.
 */
function focusElement(elem: Element) {
    const focusable = elem as Element & { focus?: () => void };
    if (typeof focusable.focus === "function") {
        focusable.focus();
    }
}

/**
 * Blurs an element only when the browser exposes a callable blur method.
 */
function blurElement(elem: Element) {
    const focusable = elem as Element & { blur?: () => void };
    if (typeof focusable.blur === "function") {
        focusable.blur();
    }
}

/**
 * Scrolls an element into view when supported by the current DOM environment.
 */
function scrollElementIntoView(elem: Element) {
    if ("scrollIntoView" in elem && typeof elem.scrollIntoView === "function") {
        elem.scrollIntoView();
    }
}

/**
 * Sets disabled state on controls that support a `.disabled` property.
 */
function setElementDisabled(elem: Element, disabled: boolean) {
    if ("disabled" in elem) {
        (elem as HTMLButtonElement | HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement).disabled = disabled;
    }
}
