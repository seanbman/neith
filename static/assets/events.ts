import { parseEventData } from "./event_payloads";
import type { Dispatch, FnEventListener } from "./fcmp_types";
import { Fun } from "./fcmp_types";
import { emitHook } from "./hooks";

export type DispatchSender = (data: Dispatch | void) => void;
export type ErrorReporter = (d: Dispatch, message: string) => void;

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

export function addEventListeners(
    d: Dispatch,
    send: DispatchSender,
    reportError: ErrorReporter
) {
    if (!d.render.event_listeners) return;

    d.render.event_listeners.forEach((listener: FnEventListener) => {
        let elem = document.getElementById(listener.target_id);
        if (!elem) {
            reportError(d, "element not found");
            return;
        }
        if (elem.firstElementChild) {
            elem = elem.firstElementChild as HTMLElement;
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
        });
    });
}

function getAttributes(elem: Element, attribute: string): string[] {
    const elems = elem.querySelectorAll(`[${attribute}]`);
    return Array.from(elems)
        .map((el) => el.getAttribute(attribute))
        .filter((value): value is string => value !== null);
}

function cloneDispatch(d: Dispatch): Dispatch {
    return JSON.parse(JSON.stringify(d)) as Dispatch;
}
