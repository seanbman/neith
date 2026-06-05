import { API } from "./api";
import { Dispatch } from "./fcmp_types";
var did_connect = false;
let api: API;

export class Socket {
    private ws: WebSocket | null = null;
    private addr: string | undefined = undefined;
    private key: string | undefined = undefined;

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
            api = new API(this.ws);
        } catch (err) {
            throw new Error("ws: failed to initiate API: " + err);
        }

        this.ws.onopen = function () {
            did_connect = true;
        };
        this.ws.onclose = function () {
            setTimeout(() => {
                if (typeof window !== "undefined")
                    window.location.reload();
            }, 1000);
        };
        this.ws.onerror = function () {};

        this.ws.onmessage = function (event) {
            let d = JSON.parse(event.data) as Dispatch;
            api.Process(d);
        };
    }
}
