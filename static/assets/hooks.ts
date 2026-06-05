import type { Dispatch } from "./fcmp_types";

export type HookName =
    | "connect"
    | "disconnect"
    | "reconnect"
    | "beforeRender"
    | "afterRender"
    | "beforeEventDispatch"
    | "afterEventDispatch"
    | "error";

export type HookPayload = {
    dispatch?: Dispatch;
    element?: Element;
    event?: Event;
    error?: string;
};

type HookCallback = (payload: HookPayload) => void;

const hooks = new Map<HookName, Set<HookCallback>>();

export function onHook(name: HookName, callback: HookCallback) {
    if (!hooks.has(name)) {
        hooks.set(name, new Set());
    }
    hooks.get(name)?.add(callback);

    return () => offHook(name, callback);
}

export function offHook(name: HookName, callback: HookCallback) {
    hooks.get(name)?.delete(callback);
}

export function emitHook(name: HookName, payload: HookPayload = {}) {
    hooks.get(name)?.forEach((callback) => callback(payload));
}

export function installHookAPI() {
    if (typeof window === "undefined") return;

    const root = ((window as unknown as WindowWithFCMP).fcmp ||= {});
    root.on = onHook;
    root.off = offHook;
    root.hooks = {
        on: onHook,
        off: offHook,
    };
}

type WindowWithFCMP = Window & {
    fcmp?: {
        on?: typeof onHook;
        off?: typeof offHook;
        hooks?: {
            on: typeof onHook;
            off: typeof offHook;
        };
    };
};

installHookAPI();
