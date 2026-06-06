import { addEventListeners, parseEventListeners } from "./events";
import type { Dispatch, DispatchFunctions } from "./neith_types";
import { Fun } from "./neith_types";
import { emitHook } from "./hooks";
import { applyClass, applyCustom, applyDOM, applyRender } from "./render";

/**
 * API owns the browser side of the neith dispatch protocol.
 *
 * The websocket receives a Dispatch object from Go, then API.Process routes that
 * dispatch to the correct DOM operation. Some operations, such as ping and
 * custom JavaScript calls, produce a response dispatch that is sent back to the
 * server over the same websocket.
 */
export class API {
    private ws: WebSocket | null = null;

    /**
     * Stores the connected websocket used for all browser-to-server replies.
     *
     * API does not open or reconnect sockets itself. Socket is responsible for
     * connection lifecycle; API only assumes it has a live WebSocket-like object
     * when it needs to send a dispatch response.
     */
    constructor(ws: WebSocket) {
        this.ws = ws;
    }

    /**
     * Routes one dispatch from the server to the matching browser operation.
     *
     * Redirects are handled directly because they intentionally leave the neith
     * page. All other dispatches go through the `funs` lookup. If the operation
     * returns another dispatch, that response is serialized back to Go.
     */
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

    /**
     * Sends a dispatch back to the server through the active websocket.
     *
     * DOM-only operations return void, so the method intentionally accepts void
     * and exits early. Any real dispatch is JSON encoded to match the Go side's
     * websocket protocol.
     */
    private Dispatch = (data: Dispatch | void) => {
        if (!data) return;
        if (!this.ws) {
            throw new Error("ws: not connected to server...");
        }
        this.ws.send(JSON.stringify(data));
    };

    /**
     * Converts a client-side failure into an neith error dispatch.
     *
     * This keeps browser failures visible to the server instead of only throwing
     * in the console. The error hook fires before the dispatch is sent so app
     * code can also observe or log browser-side failures.
     */
    private Error = (d: Dispatch, message: string) => {
        d.function = Fun.ERROR;
        d.error = { message };
        emitHook("error", { dispatch: d, error: message });
        this.Dispatch(d);
    };

    /**
     * Dispatch handlers keyed by the protocol's function name.
     *
     * Each handler performs one browser-side effect and optionally returns a
     * dispatch response. Keeping these in a table makes Process small and keeps
     * each operation's behavior isolated.
     */
    private funs: DispatchFunctions = {
        /**
         * Marks a server ping as answered by the client.
         *
         * The same dispatch object is returned so Go can see `client: true` and
         * confirm the browser is still responsive.
         */
        ping: (d: Dispatch) => {
            d.ping.client = true;
            return d;
        },
        /**
         * Applies server-rendered HTML and wires any event metadata it contains.
         *
         * Event listener attributes live inside the rendered HTML. After the DOM
         * is updated, the rendered subtree is scanned, listeners are attached,
         * and render lifecycle hooks are emitted around the operation.
         */
        render: (d: Dispatch) => {
            emitHook("beforeRender", { dispatch: d });
            const elem = applyRender(d, this.Error);
            if (!elem) return;

            const dispatch = parseEventListeners(elem, d);
            addEventListeners(dispatch, this.Dispatch, this.Error);
            emitHook("afterRender", { dispatch, element: elem });
            return;
        },
        /**
         * Adds or removes CSS classes on an existing DOM element.
         */
        class: (d: Dispatch) => {
            return applyClass(d, this.Error);
        },
        /**
         * Applies a focused DOM mutation such as setting text, value, style, or
         * focus state without replacing an element's HTML.
         */
        dom: (d: Dispatch) => {
            return applyDOM(d, this.Error);
        },
        /**
         * Calls a named browser function and returns its result to Go.
         */
        custom: (d: Dispatch) => {
            return applyCustom(d, this.Error);
        },
    };
}
