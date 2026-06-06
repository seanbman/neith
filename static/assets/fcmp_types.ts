/**
 * Lookup table for browser-side dispatch handlers.
 *
 * The key is a protocol function name, and the value either performs a DOM
 * effect or returns a response dispatch that should be sent back to Go.
 */
type DispatchFunctions = {
    [key: string]: (data: Dispatch) => Dispatch | void;
};

/**
 * Protocol function names shared by Go and the browser client.
 */
enum Fun {
    AUTH = "auth",
    KEY = "key",
    PING = "ping",
    RENDER = "render",
    CLASS = "class",
    DOM = "dom",
    CUSTOM = "custom",
    REDIRECT = "redirect",
    EVENT = "event",
    ERROR = "error",
}

/**
 * Authentication payload reserved for auth/key dispatches.
 */
type FnAuth = {
    key: string;
    token: string;
};

/**
 * Metadata for one server-side event handler attached to a rendered component.
 *
 * The Go side creates this object and serializes it into the rendered HTML. The
 * browser later sends the same metadata back with parsed event data.
 */
type FnEventListener = {
    id: string;
    target_id: string;
    on: string;
    action: string;
    method: string;
    form_data: string;
    data: Object;
    uploads?: Upload[];
    submitter?: EventTargetData | null;
};

/**
 * JSON-safe snapshot of a DOM event target.
 */
type EventTargetData = {
    id: string;
    name: string;
    classList: string[];
    tagName: string;
    innerHTML: string;
    outerHTML: string;
    value: string;
    checked: boolean;
    disabled: boolean;
    hidden: boolean;
    style: string;
    attributes: string[];
    dataset: string[];
    selectedOptions: string[];
};

/**
 * Metadata for one file uploaded before an event dispatch.
 */
type Upload = {
    id: string;
    field_name: string;
    file_name: string;
    content_type: string;
    size: number;
    path: string;
};

/**
 * Ping payload used by the server to confirm the browser websocket is alive.
 */
type FnPing = {
    server: boolean;
    client: boolean;
};

/**
 * Render payload describing how HTML should be applied to the DOM.
 */
type FnRender = {
    target_id: string;
    tag: string;
    inner: boolean;
    outer: boolean;
    append: boolean;
    prepend: boolean;
    remove: boolean;
    html: string;
    event_listeners: FnEventListener[];
};

/**
 * Class mutation payload for adding or removing CSS class names.
 */
type FnClass = {
    target_id: string;
    remove: boolean;
    names: string[];
};

/**
 * Focused DOM mutation payload for attributes, style, text, value, and state.
 */
type FnDOM = {
    target_id: string;
    operation: string;
    name: string;
    value: string;
};

/**
 * Custom browser function call payload.
 */
type FnCustom = {
    function: string;
    data: Object;
    result: Object;
};

/**
 * Browser redirect payload.
 */
type FnRedirect = {
    url: string;
};

/**
 * Error payload sent when either side cannot process a dispatch.
 */
type FnError = {
    message: string;
};

/**
 * Full websocket message exchanged between Go and the browser.
 *
 * Only one function-specific payload is normally meaningful for a given
 * dispatch, selected by the `function` field. The flat shape mirrors the Go
 * struct so JSON marshalling stays straightforward on both sides.
 */
type Dispatch = {
    function: Fun;
    id: string;
    key: string;
    conn_id: string;
    handler_id: string;
    action: string;
    label: string;
    event: FnEventListener;
    ping: FnPing;
    render: FnRender;
    class: FnClass;
    dom: FnDOM;
    redirect: FnRedirect;
    custom: FnCustom;
    error: FnError;
};

export {
    DispatchFunctions,
    Fun,
    FnAuth,
    FnPing,
    FnRender,
    FnClass,
    FnDOM,
    FnCustom,
    FnRedirect,
    FnError,
    FnEventListener,
    EventTargetData,
    Upload,
    Dispatch,
};
