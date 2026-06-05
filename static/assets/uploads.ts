import type { Upload } from "./fcmp_types";

type UploadResponse = {
    files?: Upload[];
};

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

export function formValues(form: HTMLFormElement, submitter?: HTMLElement | null) {
    const formData = newFormData(form, submitter);
    const values: Record<string, FormDataEntryValue> = {};

    formData.forEach((value, key) => {
        if (isFile(value)) return;
        values[key] = value;
    });

    return values;
}

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

function formDataConstructor(form: HTMLFormElement): FormDataConstructor {
    return (form.ownerDocument.defaultView?.FormData || FormData) as FormDataConstructor;
}

function appendSubmitterValue(formData: FormData, submitter: HTMLElement) {
    const control = submitter as HTMLButtonElement | HTMLInputElement;
    if (!control.name || !("value" in control)) return;
    formData.append(control.name, control.value);
}

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

function uploadURL(): string {
    const url = new URL(window.location.href);
    url.searchParams.set("fcmp_upload", "1");

    const key = localStorage.getItem("fcmp");
    if (key) {
        url.searchParams.set("fcmp_id", key);
    }

    return url.toString();
}

function isFile(value: FormDataEntryValue): value is File {
    return typeof value !== "string" &&
        "name" in value &&
        "size" in value;
}

type Submitter = HTMLButtonElement | HTMLInputElement;

type FormDataConstructor = {
    new(form?: HTMLFormElement, submitter?: Submitter): FormData;
};
