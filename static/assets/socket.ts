import { API } from "./api";
import { Dispatch } from "./neith_types";
import { emitHook } from "./hooks";

/**
 * Socket owns the browser's websocket connection to the neith server.
 *
 * It builds the connection URL, keeps a stable client key in localStorage, hands
 * incoming messages to API for processing, and reconnects after unexpected
 * disconnects without forcing a page reload.
 */
export class Socket {
    private ws: WebSocket | null = null;
    private addr: string | undefined = undefined;
    private key: string | undefined = undefined;
    private api: API | null = null;
    private didConnect = false;
    private reconnectAttempts = 0;
    private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    private readonly baseReconnectDelay = 500;
    private readonly maxReconnectDelay = 30000;

    /**
     * Creates and immediately connects a socket.
     *
     * Tests can pass an explicit address. In normal browser usage, the address
     * is derived from the current page path and protocol so the client connects
     * back to the same neith endpoint that served the page.
     */
    constructor(addr?: string) {
        if (addr) {
            this.addr = addr;
        } else {
            this.init();
        }
        this.connect();
    }

    /**
     * Builds the websocket address and persistent neith client key.
     *
     * The key identifies a browser tab/session to the Go side. It is stored in
     * localStorage so reconnects and reloads can resume the same server-side
     * connection state when possible.
     */
    private init() {
        let path = window.location.pathname.split("");
        let path_parsed = "";
        if (path[-1] == "/" || (path.length == 1 && path[0] == "/")) {
            path.pop();
        }
        path_parsed = path.join("");
        
        if (path_parsed == "") {
            path_parsed = "/";
        }

        let key = localStorage.getItem("neith");
        if (!key) {
            key = "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(
                /[xy]/g,
                function (c) {
                    let r = (Math.random() * 16) | 0,
                        v = c == "x" ? r : (r & 0x3) | 0x8;
                    return v.toString(16);
                }
            );
            localStorage.setItem("neith", key);
        }
        this.key = key;

        let protocol = "wss";
        if (location.protocol !== "https:") {
            protocol = "ws";
        }

        this.addr = protocol + "://" + window.location.host + path_parsed + "?neith_id=" + this.key;
    }

    /**
     * Opens the websocket and wires browser event handlers.
     *
     * A new API instance is created for each websocket because API stores the
     * socket it uses for replies. The onopen/onclose handlers also emit lifecycle
     * hooks so application code can observe connection state.
     */
    private connect() {
        try {
            this.ws = new WebSocket(this.addr);
        } catch (err) {
            throw new Error("ws: failed to connect to neith server: " + err);
        }
        try {
            this.api = new API(this.ws);
        } catch (err) {
            throw new Error("ws: failed to initiate API: " + err);
        }

        this.ws.onopen = () => {
            this.clearReconnectTimer();
            this.reconnectAttempts = 0;
            emitHook(this.didConnect ? "reconnect" : "connect");
            this.didConnect = true;
        };
        this.ws.onclose = (event) => {
            emitHook("disconnect");
            if (this.shouldReconnect(event)) {
                this.scheduleReconnect();
            }
        };
        this.ws.onerror = () => {};

        this.ws.onmessage = (event) => {
            let d = JSON.parse(event.data) as Dispatch;
            this.api?.Process(d);
        };
    }

    /**
     * Decides whether a closed socket should reconnect.
     *
     * Normal closes, such as code 1000 or 1001, are treated as intentional and
     * do not reconnect. Unexpected closes in a real browser schedule another
     * connection attempt.
     */
    private shouldReconnect(event: CloseEvent): boolean {
        return typeof window !== "undefined" &&
            !event.wasClean &&
            event.code !== 1000 &&
            event.code !== 1001;
    }

    /**
     * Schedules the next reconnect attempt using capped exponential backoff.
     *
     * Only one timer can be active at a time. The delay grows after each failed
     * close and is reset to zero attempts when a socket opens successfully.
     */
    private scheduleReconnect() {
        if (this.reconnectTimer) return;

        const delay = Math.min(
            this.baseReconnectDelay * 2 ** this.reconnectAttempts,
            this.maxReconnectDelay
        );
        this.reconnectAttempts++;
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connect();
        }, delay);
    }

    /**
     * Clears any pending reconnect timer.
     *
     * This is called when a socket opens so an old timer cannot create a second
     * connection after the current one is already healthy.
     */
    private clearReconnectTimer() {
        if (!this.reconnectTimer) return;
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
    }
}
