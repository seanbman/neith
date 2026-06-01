type DispatchFunctions = {
    [key: string]: (data: Dispatch) => Dispatch | void;
};

enum Fun {
    AUTH = "auth",
    KEY = "key",
    PING = "ping",
    RENDER = "render",
    CLASS = "class",
    CUSTOM = "custom",
    REDIRECT = "redirect",
    EVENT = "event",
    ERROR = "error",
}

type FnAuth = {
    key: string;
    token: string;
};

type FnEventListener = {
    id: string;
    target_id: string;
    on: string;
    action: string;
    method: string;
    form_data: string;
    data: Object;
};

type FnPing = {
    server: boolean;
    client: boolean;
};

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

type FnClass = {
    target_id: string;
    remove: boolean;
    names: string[];
};

type FnCustom = {
    function: string;
    data: Object;
    result: Object;
};

type FnRedirect = {
    url: string;
};

type FnError = {
    message: string;
};

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
    FnCustom,
    FnRedirect,
    FnError,
    FnEventListener,
    Dispatch,
};
