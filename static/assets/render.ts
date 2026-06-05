import type { ErrorReporter } from "./events";
import type { Dispatch } from "./fcmp_types";

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

export function applyCustom(d: Dispatch, reportError: ErrorReporter): Dispatch | void {
    const fn = (window as unknown as Record<string, (data: Object) => Object | undefined>)[d.custom.function];
    if (typeof fn !== "function") {
        return reportError(d, "custom function not found: " + d.custom.function);
    }
    d.custom.result = fn(d.custom.data);
    return d;
}

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

function setElementValue(elem: Element, value: string) {
    if ("value" in elem) {
        (elem as HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement).value = value;
    }
}

function focusElement(elem: Element) {
    if ("focus" in elem && typeof elem.focus === "function") {
        (elem as HTMLElement).focus();
    }
}

function blurElement(elem: Element) {
    if ("blur" in elem && typeof elem.blur === "function") {
        (elem as HTMLElement).blur();
    }
}

function scrollElementIntoView(elem: Element) {
    if ("scrollIntoView" in elem && typeof elem.scrollIntoView === "function") {
        elem.scrollIntoView();
    }
}

function setElementDisabled(elem: Element, disabled: boolean) {
    if ("disabled" in elem) {
        (elem as HTMLButtonElement | HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement).disabled = disabled;
    }
}
