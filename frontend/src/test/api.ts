import { vi } from "vitest";
import type { User } from "#/features/auth/session";

export type MockResponse = {
	status?: number;
	body?: unknown;
	headers?: Record<string, string>;
};

type Handler =
	| MockResponse
	| ((req: {
			body: unknown;
			init: RequestInit;
			url: URL;
	  }) => MockResponse | Promise<MockResponse>);

/** Routes keyed by "METHOD /path". */
export type Routes = Record<string, Handler>;

/** A recorded request; search is the query string including "?", or "". */
export type Call = {
	method: string;
	path: string;
	search: string;
	body: unknown;
};

export const signedOut: MockResponse = {
	status: 401,
	body: { error: { code: "unauthenticated", message: "Sign in to continue" } },
};

export function signedIn(user: Partial<User> = {}): MockResponse {
	return {
		body: {
			id: "u1",
			email: "ann@example.org",
			name: null,
			slug: "ann",
			shareFolderId: null,
			passkeyEnrollmentUntil: null,
			...user,
		},
	};
}

export function apiError(
	status: number,
	code: string,
	headers?: Record<string, string>,
): MockResponse {
	return { status, body: { error: { code, message: code } }, headers };
}

/**
 * Replaces fetch with a mock that answers from routes (signed out without
 * passkeys by default) and records every call. Unknown routes answer 404.
 */
export function mockApi(routes: Routes = {}) {
	const all: Routes = {
		"GET /api/me": signedOut,
		"GET /api/passkeys": { body: [] },
		"GET /api/dashboards": { body: [] },
		...routes,
	};
	const calls: Call[] = [];

	const fetchMock = vi.fn(
		async (input: RequestInfo | URL, init: RequestInit = {}) => {
			const method = (init.method ?? "GET").toUpperCase();
			const url = new URL(String(input), "http://localhost");
			const path = url.pathname;
			const body =
				typeof init.body === "string" ? JSON.parse(init.body) : undefined;
			calls.push({ method, path, search: url.search, body });

			const handler = all[`${method} ${path}`];
			const res: MockResponse = !handler
				? {
						status: 404,
						body: { error: { code: "not_found", message: "Not found" } },
					}
				: typeof handler === "function"
					? await handler({ body, init, url })
					: handler;
			const status = res.status ?? 200;
			return new Response(
				status === 204 || res.body === undefined
					? null
					: JSON.stringify(res.body),
				{
					status,
					headers: { "Content-Type": "application/json", ...res.headers },
				},
			);
		},
	);
	vi.stubGlobal("fetch", fetchMock);

	return {
		calls,
		/** Changes or adds routes after rendering. */
		set(more: Routes) {
			Object.assign(all, more);
		},
		callsTo(method: string, path: string) {
			return calls.filter((c) => c.method === method && c.path === path);
		},
	};
}
