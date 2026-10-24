// Client-side checks that mirror the server's (internal/favourites/validate.go),
// so forms show problems before sending. The server stays authoritative.

export const MAX_FOLDER_NAME = 100;
export const MAX_LINK_TITLE = 1000;
export const MAX_URL = 2048;
export const MAX_TAG = 100;
export const MAX_DESCRIPTION = 2048;

/** A translation key under favourites.errors. */
export type FieldError =
	| "nameRequired"
	| "nameTooLong"
	| "urlInvalid"
	| "urlTooLong"
	| "titleTooLong"
	| "tagTooLong"
	| "descriptionTooLong";

const length = (s: string) => [...s].length;

export function checkFolderName(name: string): FieldError | null {
	const trimmed = name.trim();
	if (trimmed === "") return "nameRequired";
	return length(trimmed) > MAX_FOLDER_NAME ? "nameTooLong" : null;
}

export function checkLinkTitle(title: string): FieldError | null {
	return length(title.trim()) > MAX_LINK_TITLE ? "titleTooLong" : null;
}

export function checkUrl(url: string): FieldError | null {
	const trimmed = url.trim();
	if (length(trimmed) > MAX_URL) return "urlTooLong";
	try {
		const parsed = new URL(trimmed);
		const web = parsed.protocol === "http:" || parsed.protocol === "https:";
		return web && parsed.host !== "" ? null : "urlInvalid";
	} catch {
		return "urlInvalid";
	}
}

export function checkTag(name: string): FieldError | null {
	return length(name.trim()) > MAX_TAG ? "tagTooLong" : null;
}

export function checkDescription(description: string): FieldError | null {
	return length(description.trim()) > MAX_DESCRIPTION
		? "descriptionTooLong"
		: null;
}

/** Maps an API error code to a field error, or null for other errors. */
export function fieldErrorFromCode(
	code: string,
	kind: "folder" | "link",
): FieldError | null {
	switch (code) {
		case "invalid_title":
			return kind === "folder" ? "nameTooLong" : "titleTooLong";
		case "invalid_url":
			return "urlInvalid";
		case "invalid_tag":
			return "tagTooLong";
		case "invalid_description":
			return "descriptionTooLong";
		default:
			return null;
	}
}
