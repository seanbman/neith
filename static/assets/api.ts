import { addEventListeners, parseEventListeners } from "./events";
import type { Dispatch, DispatchFunctions } from "./fcmp_types";
import { Fun } from "./fcmp_types";
import { emitHook } from "./hooks";
import { applyClass, applyCustom, applyDOM, applyRender } from "./render";

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

    private Error = (d: Dispatch, message: string) => {
        d.function = Fun.ERROR;
        d.error = { message };
        emitHook("error", { dispatch: d, error: message });
        this.Dispatch(d);
    };

    private funs: DispatchFunctions = {
        ping: (d: Dispatch) => {
            d.ping.client = true;
            return d;
        },
        render: (d: Dispatch) => {
            emitHook("beforeRender", { dispatch: d });
            const elem = applyRender(d, this.Error);
            if (!elem) return;

            const dispatch = parseEventListeners(elem, d);
            addEventListeners(dispatch, this.Dispatch, this.Error);
            emitHook("afterRender", { dispatch, element: elem });
            return;
        },
        class: (d: Dispatch) => {
            return applyClass(d, this.Error);
        },
        dom: (d: Dispatch) => {
            return applyDOM(d, this.Error);
        },
        custom: (d: Dispatch) => {
            return applyCustom(d, this.Error);
        },
    };
}
