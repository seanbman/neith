import { API } from "./api";
import { Dispatch } from "./fcmp_types";
import { emitHook } from "./hooks";

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

    constructor(addr?: string) {
        if (addr) {
            this.addr = addr;
        } else {
            this.init();
        }
        this.connect();
    }

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

        let key = localStorage.getItem("fcmp");
        if (!key) {
            key = "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(
                /[xy]/g,
                function (c) {
                    let r = (Math.random() * 16) | 0,
                        v = c == "x" ? r : (r & 0x3) | 0x8;
                    return v.toString(16);
                }
            );
            localStorage.setItem("fcmp", key);
        }
        this.key = key;

        let protocol = "wss";
        if (location.protocol !== "https:") {
            protocol = "ws";
        }

        this.addr = protocol + "://" + window.location.host + path_parsed + "?fcmp_id=" + this.key;
    }

    private connect() {
        try {
            this.ws = new WebSocket(this.addr);
        } catch (err) {
            throw new Error("ws: failed to connect to fcmp server: " + err);
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

    private shouldReconnect(event: CloseEvent): boolean {
        return typeof window !== "undefined" &&
            !event.wasClean &&
            event.code !== 1000 &&
            event.code !== 1001;
    }

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

    private clearReconnectTimer() {
        if (!this.reconnectTimer) return;
        clearTimeout(this.reconnectTimer);
        this.reconnectTimer = null;
    }
}
