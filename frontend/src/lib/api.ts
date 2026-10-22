/** An error response from the API, parsed from `{"error": {"code", "message"}}`. */
export class ApiError extends Error {
	readonly status: number;
	readonly code: string;
	/** Seconds from the `Retry-After` header, if the response had one. */
	readonly retryAfter?: number;

	constructor(
		status: number,
		code: string,
		message: string,
		retryAfter?: number,
	) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.code = code;
		this.retryAfter = retryAfter;
	}
}

let onUnauthorized: () => void = () => {};

/** Sets what happens when any API request returns 401 (switch to signed out). */
export function setUnauthorizedHandler(handler: () => void) {
	onUnauthorized = handler;
}

type ApiInit = Omit<RequestInit, "body"> & { json?: unknown; file?: Blob };

/**
 * Calls the API. It sends a JSON content type, or `application/octet-stream`
 * with a raw `file` body (the server's cross-site check requires one of them
 * on every state-changing request), returns the parsed JSON body (undefined
 * for 204) and throws an ApiError for error responses.
 */
export async function api<T>(path: string, init: ApiInit = {}): Promise<T> {
	const { json, file, ...rest } = init;
	const headers = new Headers(rest.headers);
	headers.set(
		"Content-Type",
		file ? "application/octet-stream" : "application/json",
	);
	headers.set("Accept", "application/json");

	const res = await fetch(path, {
		...rest,
		headers,
		credentials: "same-origin",
		body: file ?? (json === undefined ? undefined : JSON.stringify(json)),
	});

	if (!res.ok) {
		const error = await parseError(res);
		if (res.status === 401) onUnauthorized();
		throw error;
	}
	if (res.status === 204) return undefined as T;
	const text = await res.text();
	return (text ? JSON.parse(text) : undefined) as T;
}

async function parseError(res: Response): Promise<ApiError> {
	let code = "error";
	let message = res.statusText || `HTTP ${res.status}`;
	try {
		const body = await res.json();
		if (typeof body?.error?.code === "string") code = body.error.code;
		if (typeof body?.error?.message === "string") message = body.error.message;
	} catch {
		// Not the API's JSON error format (e.g. a proxy error page).
	}
	const retryHeader = Number.parseInt(res.headers.get("Retry-After") ?? "", 10);
	return new ApiError(
		res.status,
		code,
		message,
		Number.isFinite(retryHeader) ? retryHeader : undefined,
	);
}
