import { parseEventData } from "./event_payloads";
import type { Dispatch, FnEventListener } from "./fcmp_types";
import { Fun } from "./fcmp_types";
import { emitHook } from "./hooks";

/**
 * Sends an optional dispatch back to the server.
 *
 * API owns the actual websocket write, while this module only needs a callback
 * so event code can remain independent from the websocket implementation.
 */
export type DispatchSender = (data: Dispatch | void) => void;

/**
 * Reports a browser-side problem as an fcmp error dispatch.
 */
export type ErrorReporter = (d: Dispatch, message: string) => void;

/**
 * Extracts fcmp event listener metadata from newly rendered HTML.
 *
 * Go serializes listener definitions into `events` attributes on component
 * wrappers. After render, this scans the rendered subtree, parses each JSON
 * attribute, and stores the flattened listener list on the dispatch so
 * addEventListeners can bind the browser events.
 */
export function parseEventListeners(element: Element, d: Dispatch): Dispatch {
    const events = getAttributes(element, "events");
    const listeners = events.map((e) => {
        const event = JSON.parse(e);
        if (!event) return;
        return event as FnEventListener[];
    });
    d.render.event_listeners = listeners.flat().filter((e) => e != null);
    return d;
}

/**
 * Binds all event listeners described by a render dispatch.
 *
 * Each listener is attached to the fcmp wrapper element identified by target_id.
 * Browser events bubble up from the actual clicked/typed/touched element, so
 * event payload parsing can include both the wrapper (`currentTarget`) and the
 * concrete interacted element (`target`).
 */
export function addEventListeners(
    d: Dispatch,
    send: DispatchSender,
    reportError: ErrorReporter
) {
    if (!d.render.event_listeners) return;

    d.render.event_listeners.forEach((listener: FnEventListener) => {
        const elem = document.getElementById(listener.target_id);
        if (!elem) {
            reportError(d, "element not found");
            return;
        }
        elem.addEventListener(listener.on, async (ev) => {
            ev.preventDefault();
            const eventDispatch = cloneDispatch(d);
            eventDispatch.function = Fun.EVENT;
            eventDispatch.event = { ...listener };
            try {
                const parsed = await parseEventData(listener.on, ev);
                eventDispatch.event.data = parsed.data || {};
                eventDispatch.event.uploads = parsed.uploads;
                eventDispatch.event.submitter = parsed.submitter;
            } catch (err) {
                reportError(d, err instanceof Error ? err.message : String(err));
                return;
            }
            emitHook("beforeEventDispatch", { dispatch: eventDispatch, event: ev });
            send(eventDispatch);
            emitHook("afterEventDispatch", { dispatch: eventDispatch, event: ev });
        }, eventOptions(listener.on));
    });
}

/**
 * Chooses listener options for events that do not bubble normally.
 *
 * Most events can be caught on the wrapper through bubbling. Focus, blur, and a
 * few related events need capture mode so the wrapper can still observe child
 * element activity.
 */
function eventOptions(eventName: string): AddEventListenerOptions {
    if (["blur", "focus", "invalid", "mouseenter", "mouseleave"].includes(eventName)) {
        return { capture: true };
    }
    return {};
}

/**
 * Returns raw attribute values for every descendant that has the given attribute.
 */
function getAttributes(elem: Element, attribute: string): string[] {
    const elems = elem.querySelectorAll(`[${attribute}]`);
    return Array.from(elems)
        .map((el) => el.getAttribute(attribute))
        .filter((value): value is string => value !== null);
}

/**
 * Deep-copies a dispatch before mutating it for an outgoing event message.
 *
 * Render dispatches are reused as the template for event dispatches. Cloning
 * prevents one browser event from mutating the original render dispatch or
 * leaking state into another event.
 */
function cloneDispatch(d: Dispatch): Dispatch {
    return JSON.parse(JSON.stringify(d)) as Dispatch;
}
