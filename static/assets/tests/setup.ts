import WS from "jest-websocket-mock";
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

// Historical integration fixtures still construct the internal Dispatch shape.
// Normalize only server -> browser traffic to the public v1 envelope. Browser ->
// server assertions decode the actual wire message inside the test itself, so
// both directions cross the same codec boundary exercised by production code.
const originalServerSend = WS.prototype.send;
WS.prototype.send = function (data: unknown) {
    return originalServerSend.call(this, toProtocolFrame(data) as never);
};
