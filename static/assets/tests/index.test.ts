import WS from "jest-websocket-mock";
import { Socket } from "../socket";
import {
    describe,
    beforeAll,
    test,
    afterAll,
    expect,
    beforeEach,
    jest,
} from "@jest/globals";
import { JSDOM } from "jsdom";
import { Dispatch, Fun } from "../neith_types";
import { onHook } from "../hooks";

describe("test websocket functions", () => {
    let dispatches: Dispatch[] = [];
    let server: WS;
    let socket: Socket;

    // Wait for a callback to return true
    async function waitCallback(callback: () => boolean) {
        return new Promise((resolve) => {
            const check = () => {
                setTimeout(() => {
                    if (callback()) {
                        resolve(null);
                    } else {
                        check();
                    }
                }, 25);
            };
            check();
        });
    }

    beforeAll(async () => {
        // Create a new websocket server
        server = new WS("ws://localhost:1234", { jsonProtocol: true });
        server.on("connection", (socket) => {
            // When the server receives a message, parse it and add it to the dispatches array
            socket.on("message", (message) => {
                console.log("server received message from API: ", message.toString());
                const msg: Dispatch = JSON.parse(message.toString());
                dispatches.push(msg);
            });
        });
        // Create a new Socket client
        socket = new Socket("ws://localhost:1234");
        await server.connected;
    });

    beforeEach(async () => {
        // Wait for api to process previous dispatches
        await new Promise((resolve) => setTimeout(resolve, 500));
        dispatches = [];
        const jsdom = new JSDOM(
            "<!DOCTYPE html><html><body><main><p>test</p></main></body></html>"
        );
        global.document = jsdom.window.document;
    });

    test("test ping", async () => {
        const dispatch = {
            function: Fun.PING,
            ping: {
                server: true,
                client: false,
            },
        };
        server.send(dispatch);
        await waitCallback(() => dispatches.length > 0);
        expect(dispatches.length).toEqual(1);
        expect(dispatches[0].ping.client).toEqual(true);
    });

    const test_cases = [
        {
            tag: "main",
            name: "test render inner",
            html: "<p>test render inner</p>",
            inner: true,
            expected: "<p>test render inner</p>",
        },
        {
            tag: "main",
            name: "test render append",
            html: "<p>test render append</p>",
            append: true,
            expected: "<p>test</p><p>test render append</p>",
        },
        {
            tag: "main",
            name: "test render prepend",
            html: "<p>test render prepend</p>",
            prepend: true,
            expected: "<p>test render prepend</p><p>test</p>",
        },
        {
            tag: "main",
            name: "test render outer",
            html: "<p>test</p>",
            outer: true,
            expected: "",
        },
        {
            tag: "main",
            name: "test render remove",
            html: "",
            remove: true,
            expected: "",
        },
    ];

    test_cases.forEach((test_case) => {
        test(test_case.name, async () => {
            const dispatch = {
                function: Fun.RENDER,
                render: {
                    tag: test_case.tag,
                    html: test_case.html,
                    inner: test_case.inner,
                    append: test_case.append,
                    prepend: test_case.prepend,
                    outer: test_case.outer,
                    remove: test_case.remove,
                },
            } as Dispatch;
            server.send(dispatch);

            let html: string = "";
            await waitCallback(() => {
                if (dispatch.render.tag !== "")
                    html =
                        document.querySelector(dispatch.render.tag)
                            ?.innerHTML ?? "";    
                else
                    html =
                        document.getElementById(dispatch.render.target_id)
                            ?.innerHTML ?? "";
                return html === test_case.expected;
            });
            expect(html).toEqual(test_case.expected);
        });
    });

    test("test event listener", async () => {
        // Setup on focus event listener for a button
        const event = {
            id: "_",
            target_id: "test",
            on: "focus",
            action: "test",
            method: "GET",
            form_data: "",
            data: {},
        };
        const dispatch = {
            function: Fun.RENDER,
            render: {
                tag: "main",
                html: `<div id="${event.target_id}" events=[${JSON.stringify(
                    event
                )}]><button id="test_button">Test</button></div>`,
                append: true,
            },
        };

        server.send(dispatch);
        await waitCallback(() => {
            const elem = document.querySelector("button");
            if(!elem) return false;
            elem.focus();
            return true;
        });
        await waitCallback(() => dispatches.length > 0);
        expect(dispatches[0].event.target_id).toEqual(event.target_id);
    });

    test("test rich event component and source metadata", async () => {
        const event = {
            id: "rich-event",
            target_id: "rich-target",
            on: "click",
            action: "test",
            method: "GET",
            form_data: "",
            data: {},
        };
        server.send({
            function: Fun.RENDER,
            render: {
                tag: "main",
                html: `<div id="${event.target_id}" events=[${JSON.stringify(event)}]>
                    <button id="rich-button" name="status" class="primary important" data-role="admin" style="color: red" value="open">Open</button>
                </div>`,
                append: true,
            },
        });

        await waitCallback(() => document.getElementById("rich-button") !== null);
        const click = new document.defaultView!.MouseEvent("click", { bubbles: true, cancelable: true });
        document.getElementById("rich-button")?.dispatchEvent(click);
        await waitCallback(() => dispatches.length > 0);

        const component = (dispatches[0].event.data as any).component;
        const source = (dispatches[0].event.data as any).source;
        expect(component.id).toEqual("rich-target");
        expect(source.id).toEqual("rich-button");
        expect(source.name).toEqual("status");
        expect(source.classList).toEqual(["primary", "important"]);
        expect(source.dataset).toEqual(["role=admin"]);
        expect(source.disabled).toEqual(false);
        expect(source.attributes).toContain("value=open");
    });

    test("test keyboard event component and source metadata", async () => {
        const event = {
            id: "keyboard-event",
            target_id: "keyboard-target",
            on: "keydown",
            action: "test",
            method: "GET",
            form_data: "",
            data: {},
        };
        server.send({
            function: Fun.RENDER,
            render: {
                tag: "main",
                html: `<div id="${event.target_id}" events=[${JSON.stringify(event)}]><input id="keyboard-input" name="search" value="pipe"></div>`,
                append: true,
            },
        });

        await waitCallback(() => document.getElementById("keyboard-input") !== null);
        const keydown = new document.defaultView!.KeyboardEvent("keydown", {
            bubbles: true,
            cancelable: true,
            key: "Enter",
            code: "Enter",
        });
        document.getElementById("keyboard-input")?.dispatchEvent(keydown);
        await waitCallback(() => dispatches.length > 0);

        const data = dispatches[0].event.data as any;
        expect(data.key).toEqual("Enter");
        expect(data.code).toEqual("Enter");
        expect(data.component.id).toEqual("keyboard-target");
        expect(data.source.id).toEqual("keyboard-input");
        expect(data.source.value).toEqual("pipe");
    });

    test("test drag event component and source metadata", async () => {
        const event = {
            id: "drag-event",
            target_id: "drag-target",
            on: "dragstart",
            action: "test",
            method: "GET",
            form_data: "",
            data: {},
        };
        server.send({
            function: Fun.RENDER,
            render: {
                tag: "main",
                html: `<div id="${event.target_id}" events=[${JSON.stringify(event)}]><button id="drag-button" draggable="true">Drag</button></div>`,
                append: true,
            },
        });

        await waitCallback(() => document.getElementById("drag-button") !== null);
        const dragstart = new document.defaultView!.MouseEvent("dragstart", {
            bubbles: true,
            cancelable: true,
            clientX: 10,
            clientY: 20,
        });
        document.getElementById("drag-button")?.dispatchEvent(dragstart);
        await waitCallback(() => dispatches.length > 0);

        const data = dispatches[0].event.data as any;
        expect(data.clientX).toEqual(10);
        expect(data.clientY).toEqual(20);
        expect(data.component.id).toEqual("drag-target");
        expect(data.source.id).toEqual("drag-button");
    });

    test("test touch event component and source metadata", async () => {
        const event = {
            id: "touch-event",
            target_id: "touch-target",
            on: "touchstart",
            action: "test",
            method: "GET",
            form_data: "",
            data: {},
        };
        server.send({
            function: Fun.RENDER,
            render: {
                tag: "main",
                html: `<div id="${event.target_id}" events=[${JSON.stringify(event)}]><button id="touch-button">Touch</button></div>`,
                append: true,
            },
        });

        await waitCallback(() => document.getElementById("touch-button") !== null);
        const touchButton = document.getElementById("touch-button") as HTMLButtonElement;
        const touch = {
            clientX: 11,
            clientY: 22,
            identifier: 1,
            pageX: 33,
            pageY: 44,
            radiusX: 5,
            radiusY: 6,
            rotationAngle: 0,
            screenX: 55,
            screenY: 66,
            target: touchButton,
        };
        const touchstart = new document.defaultView!.Event("touchstart", { bubbles: true, cancelable: true });
        Object.defineProperties(touchstart, {
            changedTouches: { value: [touch] },
            targetTouches: { value: [touch] },
            touches: { value: [touch] },
            layerX: { value: 1 },
            layerY: { value: 2 },
            pageX: { value: 33 },
            pageY: { value: 44 },
        });
        touchButton.dispatchEvent(touchstart);
        await waitCallback(() => dispatches.length > 0);

        const data = dispatches[0].event.data as any;
        expect(data.component.id).toEqual("touch-target");
        expect(data.source.id).toEqual("touch-button");
        expect(data.changedTouches[0].source.id).toEqual("touch-button");
        expect(data.touches[0].clientX).toEqual(11);
        expect(data.layerX).toEqual(1);
    });

    test("test form submitter metadata", async () => {
        const event = {
            id: "submitter-event",
            target_id: "submitter-target",
            on: "submit",
            action: "test",
            method: "POST",
            form_data: "",
            data: {},
        };
        server.send({
            function: Fun.RENDER,
            render: {
                tag: "main",
                html: `<div id="${event.target_id}" events=[${JSON.stringify(event)}]><form>
                    <input name="title" value="Invoice">
                    <button id="save" name="intent" value="save" data-action="save">Save</button>
                    <button id="delete" name="intent" value="delete">Delete</button>
                </form></div>`,
                append: true,
            },
        });

        await waitCallback(() => document.getElementById("save") !== null);
        const form = document.querySelector("form") as HTMLFormElement;
        const saveButton = document.getElementById("save") as HTMLButtonElement;
        const SubmitEventCtor = (document.defaultView as any).SubmitEvent;
        const submit = SubmitEventCtor
            ? new SubmitEventCtor("submit", { bubbles: true, cancelable: true, submitter: saveButton })
            : new document.defaultView!.Event("submit", { bubbles: true, cancelable: true });
        if (!("submitter" in submit)) {
            Object.defineProperty(submit, "submitter", { value: saveButton });
        }

        form.dispatchEvent(submit);
        await waitCallback(() => dispatches.length > 0);

        expect(dispatches[0].event.data).toEqual({ title: "Invoice", intent: "save" });
        expect(dispatches[0].event.submitter?.id).toEqual("save");
        expect(dispatches[0].event.submitter?.name).toEqual("intent");
        expect(dispatches[0].event.submitter?.value).toEqual("save");
        expect(dispatches[0].event.submitter?.dataset).toEqual(["action=save"]);
    });

    test("test lifecycle hooks", async () => {
        const seen: string[] = [];
        const offBeforeRender = onHook("beforeRender", () => seen.push("beforeRender"));
        const offAfterRender = onHook("afterRender", () => seen.push("afterRender"));
        const offBeforeEvent = onHook("beforeEventDispatch", () => seen.push("beforeEventDispatch"));
        const offAfterEvent = onHook("afterEventDispatch", () => seen.push("afterEventDispatch"));

        const event = {
            id: "hook-event",
            target_id: "hook-target",
            on: "click",
            action: "test",
            method: "GET",
            form_data: "",
            data: {},
        };
        server.send({
            function: Fun.RENDER,
            render: {
                tag: "main",
                html: `<div id="${event.target_id}" events=[${JSON.stringify(event)}]><button>Hook</button></div>`,
                append: true,
            },
        });

        await waitCallback(() => seen.includes("afterRender"));
        const mouseEvent = new document.defaultView!.MouseEvent("click", { bubbles: true, cancelable: true });
        document.querySelector("button")?.dispatchEvent(mouseEvent);
        await waitCallback(() => seen.includes("afterEventDispatch"));

        expect(seen).toEqual([
            "beforeRender",
            "afterRender",
            "beforeEventDispatch",
            "afterEventDispatch",
        ]);

        offBeforeRender();
        offAfterRender();
        offBeforeEvent();
        offAfterEvent();
    });

    test("test socket reconnects after unexpected disconnect", async () => {
        const jsdom = new JSDOM(
            "<!DOCTYPE html><html><body></body></html>",
            { url: "http://localhost/reconnect-test" }
        );
        (global as any).window = jsdom.window;

        const seen: string[] = [];
        const offConnect = onHook("connect", () => seen.push("connect"));
        const offDisconnect = onHook("disconnect", () => seen.push("disconnect"));
        const offReconnect = onHook("reconnect", () => seen.push("reconnect"));

        let reconnectServer = new WS("ws://localhost:1235", { jsonProtocol: true });
        const reconnectSocket = new Socket("ws://localhost:1235");
        await reconnectServer.connected;
        await waitCallback(() => seen.includes("connect"));

        reconnectServer.error({ wasClean: false, code: 1003, reason: "boom" });
        await reconnectServer.closed;

        reconnectServer = new WS("ws://localhost:1235", { jsonProtocol: true });
        await reconnectServer.connected;
        await waitCallback(() => seen.includes("reconnect"));

        expect(reconnectSocket).toBeDefined();
        expect(seen).toEqual(["connect", "disconnect", "reconnect"]);

        reconnectServer.close({ wasClean: true, code: 1000, reason: "done" });
        await reconnectServer.closed;
        offConnect();
        offDisconnect();
        offReconnect();
    });

    test("test file upload event metadata", async () => {
        const jsdom = new JSDOM(
            "<!DOCTYPE html><html><body><main></main></body></html>",
            { url: "http://localhost/upload-test" }
        );
        global.document = jsdom.window.document;
        (global as any).window = jsdom.window;
        (global as any).localStorage = jsdom.window.localStorage;
        (global as any).FormData = jsdom.window.FormData;
        localStorage.setItem("neith", "upload-key");

        const uploadedFile = {
            id: "upload-1",
            field_name: "avatar",
            file_name: "hello.txt",
            content_type: "text/plain",
            size: 5,
            path: "/tmp/neith/hello.txt",
        };
        (global as any).fetch = jest.fn(async () => ({
            ok: true,
            json: async () => ({ files: [uploadedFile] }),
        }));

        const event = {
            id: "upload-event",
            target_id: "upload-target",
            on: "submit",
            action: "test",
            method: "POST",
            form_data: "",
            data: {},
        };
        server.send({
            function: Fun.RENDER,
            render: {
                tag: "main",
                html: `<div id="${event.target_id}" events=[${JSON.stringify(event)}]><form>
                        <input name="title" value="Profile">
                        <input id="avatar" type="file" name="avatar">
                        <button>Save</button>
                    </form></div>`,
                append: true,
            },
        });

        await waitCallback(() => document.querySelector("form") !== null);
        const input = document.getElementById("avatar") as HTMLInputElement;
        const file = new jsdom.window.File(["hello"], "hello.txt", { type: "text/plain" });
        Object.defineProperty(input, "files", {
            value: [file],
        });

        document.querySelector("form")?.dispatchEvent(new jsdom.window.Event("submit", { bubbles: true, cancelable: true }));
        await waitCallback(() => dispatches.length > 0);

        expect((global as any).fetch).toHaveBeenCalledTimes(1);
        expect(dispatches[0].event.data).toEqual({ title: "Profile" });
        expect(dispatches[0].event.uploads).toEqual([uploadedFile]);
    });

    test("test error", async () => {
        const dispatch = {
            function: Fun.RENDER,
            render: {
                tag: "test",
                html: "<p>test</p>",
                inner: true,
            }
        };
        server.send(dispatch);
        await waitCallback(() => dispatches.length > 0);
        expect(dispatches[0].error.message).toBeDefined();
    });

    test("test dom operations", async () => {
        document.body.innerHTML = `
            <main>
                <input id="field" value="old">
                <button id="button">Button</button>
                <p id="label">Old text</p>
            </main>
        `;

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "field",
                operation: "setAttribute",
                name: "aria-label",
                value: "Name",
            },
        });
        await waitCallback(() => document.getElementById("field")?.getAttribute("aria-label") === "Name");

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "field",
                operation: "removeAttribute",
                name: "aria-label",
            },
        });
        await waitCallback(() => !document.getElementById("field")?.hasAttribute("aria-label"));

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "label",
                operation: "setStyle",
                name: "color",
                value: "red",
            },
        });
        await waitCallback(() => (document.getElementById("label") as HTMLElement).style.color === "red");

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "label",
                operation: "removeStyle",
                name: "color",
            },
        });
        await waitCallback(() => (document.getElementById("label") as HTMLElement).style.color === "");

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "label",
                operation: "setText",
                value: "New text",
            },
        });
        await waitCallback(() => document.getElementById("label")?.textContent === "New text");

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "field",
                operation: "setValue",
                value: "new",
            },
        });
        await waitCallback(() => (document.getElementById("field") as HTMLInputElement).value === "new");

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "field",
                operation: "focus",
            },
        });
        await waitCallback(() => document.activeElement?.id === "field");

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "field",
                operation: "blur",
            },
        });
        await waitCallback(() => document.activeElement?.id !== "field");

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "button",
                operation: "disable",
            },
        });
        await waitCallback(() => (document.getElementById("button") as HTMLButtonElement).disabled);

        server.send({
            function: Fun.DOM,
            dom: {
                target_id: "button",
                operation: "enable",
            },
        });
        await waitCallback(() => !(document.getElementById("button") as HTMLButtonElement).disabled);
    });

    afterAll(() => {
        delete (global as any).window;
        delete (global as any).localStorage;
        delete (global as any).FormData;
        delete (global as any).fetch;
        WS.clean();
    });
});
