import WS from "jest-websocket-mock";
import { decodeMessage } from "../protocol";
import { Fun, PROTOCOL_VERSION } from "../neith_types";

const metadataKeys = ["id", "key", "conn_id", "handler_id", "action", "label"] as const;

function toProtocolFrame(data: unknown): unknown {
    if (data === null || typeof data !== "object" || Array.isArray(data)) return data;

    const record = data as Record<string, unknown>;
    if (typeof record.type === "string") {
        return "v" in record ? record : { v: PROTOCOL_VERSION, ...record };
    }

    if (typeof record.function !== "string") return data;

    const type = record.function as Fun;
    const frame: Record<string, unknown> = {
        v: PROTOCOL_VERSION,
        type,
        payload: record[type],
    };

    for (const key of metadataKeys) {
        if (typeof record[key] === "string" && record[key] !== "") frame[key] = record[key];
    }

    return frame;
}

// Most integration fixtures still construct the historical internal Dispatch shape.
// Normalize server -> browser traffic to the real v1 wire envelope so every frame
// enters production code through protocol.decodeMessage(...).
const originalServerSend = WS.prototype.send;
WS.prototype.send = function (data: unknown) {
    return originalServerSend.call(this, toProtocolFrame(data) as never);
};

// The older integration assertions inspect internal Dispatch values on the mock
// server. Preserve those assertions during Phase 2 while validating the outbound
// v1 frame before converting it back to the internal representation for the test.
const originalClientSend = globalThis.WebSocket.prototype.send;
globalThis.WebSocket.prototype.send = function (data: string | ArrayBufferLike | Blob | ArrayBufferView) {
    if (typeof data !== "string") return originalClientSend.call(this, data);

    try {
        const dispatch = decodeMessage(JSON.parse(data));
        return originalClientSend.call(this, JSON.stringify(dispatch));
    } catch {
        return originalClientSend.call(this, data);
    }
};
