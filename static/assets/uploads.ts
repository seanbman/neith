import type { Upload } from "./fcmp_types";

/**
 * JSON response returned by fcmp's HTTP upload endpoint.
 */
type UploadResponse = {
    files?: Upload[];
};

/**
 * Uploads file inputs from a form before the websocket event is dispatched.
 *
 * File bytes are sent over HTTP because websocket event payloads should stay
 * small JSON messages. The server responds with Upload metadata, which is then
 * attached to the event dispatch for Go handlers to read with EventUploads.
 */
export async function uploadFormFiles(form: HTMLFormElement): Promise<Upload[]> {
    const files = collectFiles(form);
    if (files.length === 0) return [];

    const body = new FormData();
    files.forEach(({ name, file }) => {
        body.append(name, file, file.name);
    });

    const response = await fetch(uploadURL(), {
        method: "POST",
        body,
    });
    if (!response.ok) {
        throw new Error("file upload failed: " + response.statusText);
    }

    const result = await response.json() as UploadResponse;
    return result.files || [];
}

/**
 * Reads normal form values into a plain object.
 *
 * File values are intentionally skipped because they are uploaded separately.
 * When a submitter button is known, its name/value pair is included so Go can
 * tell which submit action the user chose.
 */
export function formValues(form: HTMLFormElement, submitter?: HTMLElement | null) {
    const formData = newFormData(form, submitter);
    const values: Record<string, FormDataEntryValue> = {};

    formData.forEach((value, key) => {
        if (isFile(value)) return;
        values[key] = value;
    });

    return values;
}

/**
 * Creates FormData using the form's owning window when possible.
 *
 * In browsers this is effectively `new FormData(form, submitter)`. In tests,
 * the DOM may come from jsdom while the global FormData comes from Node, so the
 * form's own constructor avoids cross-environment type issues.
 */
function newFormData(form: HTMLFormElement, submitter?: HTMLElement | null) {
    const FormDataCtor = formDataConstructor(form);
    if (!submitter) {
        return new FormDataCtor(form);
    }

    try {
        return new FormDataCtor(form, submitter as Submitter);
    } catch {
        const formData = new FormDataCtor(form);
        appendSubmitterValue(formData, submitter);
        return formData;
    }
}

/**
 * Returns the FormData constructor that belongs to the form's document.
 */
function formDataConstructor(form: HTMLFormElement): FormDataConstructor {
    return (form.ownerDocument.defaultView?.FormData || FormData) as FormDataConstructor;
}

/**
 * Adds the clicked submitter's name/value pair when FormData lacks native support.
 *
 * Older DOM implementations may not support `new FormData(form, submitter)`.
 * This fallback preserves the important behavior manually.
 */
function appendSubmitterValue(formData: FormData, submitter: HTMLElement) {
    const control = submitter as HTMLButtonElement | HTMLInputElement;
    if (!control.name || !("value" in control)) return;
    formData.append(control.name, control.value);
}

/**
 * Finds all selected files in named file inputs.
 *
 * Empty placeholder File objects are skipped so untouched file inputs do not
 * create meaningless uploads.
 */
function collectFiles(form: HTMLFormElement): Array<{ name: string; file: File }> {
    const files: Array<{ name: string; file: File }> = [];

    Array.from(form.elements).forEach((element) => {
        const input = element as HTMLInputElement;
        if (input.tagName !== "INPUT" || input.type !== "file") {
            return;
        }
        if (!input.name || !input.files) {
            return;
        }
        Array.from(input.files).forEach((file) => {
            if (file.size === 0 && file.name === "") return;
            files.push({ name: input.name, file });
        });
    });

    return files;
}

/**
 * Builds the upload endpoint URL for the current page.
 *
 * The fcmp ID is included when present so the Go side can associate the upload
 * request with the same connection/session as the later websocket event.
 */
function uploadURL(): string {
    const url = new URL(window.location.href);
    url.searchParams.set("fcmp_upload", "1");

    const key = localStorage.getItem("fcmp");
    if (key) {
        url.searchParams.set("fcmp_id", key);
    }

    return url.toString();
}

/**
 * Distinguishes real file entries from string form fields.
 */
function isFile(value: FormDataEntryValue): value is File {
    return typeof value !== "string" &&
        "name" in value &&
        "size" in value;
}

/**
 * Form controls that may be used as form submitters.
 */
type Submitter = HTMLButtonElement | HTMLInputElement;

/**
 * FormData constructor shape used by browsers that support submitter arguments.
 */
type FormDataConstructor = {
    new(form?: HTMLFormElement, submitter?: Submitter): FormData;
};
