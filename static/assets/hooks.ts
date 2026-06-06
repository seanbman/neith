import type { Dispatch } from "./fcmp_types";

/**
 * Names of browser lifecycle hooks exposed by the fcmp client.
 */
export type HookName =
    | "connect"
    | "disconnect"
    | "reconnect"
    | "beforeRender"
    | "afterRender"
    | "beforeEventDispatch"
    | "afterEventDispatch"
    | "error";

/**
 * Context passed to hook callbacks.
 *
 * Different hooks populate different fields. Render hooks include dispatch and
 * element details, event hooks include the native Event, and error hooks include
 * a message string.
 */
export type HookPayload = {
    dispatch?: Dispatch;
    element?: Element;
    event?: Event;
    error?: string;
};

type HookCallback = (payload: HookPayload) => void;

const hooks = new Map<HookName, Set<HookCallback>>();

/**
 * Registers a callback for one fcmp lifecycle hook.
 *
 * The returned function removes the callback, which is useful for tests and for
 * application code that mounts/unmounts UI around fcmp.
 */
export function onHook(name: HookName, callback: HookCallback) {
    if (!hooks.has(name)) {
        hooks.set(name, new Set());
    }
    hooks.get(name)?.add(callback);

    return () => offHook(name, callback);
}

/**
 * Removes a previously registered hook callback.
 */
export function offHook(name: HookName, callback: HookCallback) {
    hooks.get(name)?.delete(callback);
}

/**
 * Runs all callbacks registered for a hook name.
 *
 * The optional payload defaults to an empty object so emitters can fire simple
 * lifecycle notifications without building unnecessary data.
 */
export function emitHook(name: HookName, payload: HookPayload = {}) {
    hooks.get(name)?.forEach((callback) => callback(payload));
}

/**
 * Installs the public hook API on window.fcmp.
 *
 * This gives application JavaScript a stable way to subscribe to fcmp lifecycle
 * events without importing this module directly from the bundle.
 */
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

/**
 * Shape added to the browser window by installHookAPI.
 */
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
