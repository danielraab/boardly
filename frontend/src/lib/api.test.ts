import { describe, expect, it, vi } from "vitest";
import { createQueryClient, meQueryKey } from "#/features/auth/session";
import { apiError, mockApi, signedIn } from "#/test/api";
import { ApiError, api } from "./api";

describe("api", () => {
	it("sends JSON with the JSON content type and parses the response", async () => {
		const mock = mockApi({ "PATCH /api/me": signedIn({ name: "Ann" }) });
		const user = await api<{ name: string }>("/api/me", {
			method: "PATCH",
			json: { name: "Ann" },
		});
		expect(user.name).toBe("Ann");
		expect(mock.callsTo("PATCH", "/api/me")[0].body).toEqual({ name: "Ann" });

		const init = vi.mocked(fetch).mock.calls[0][1] as RequestInit;
		expect(new Headers(init.headers).get("Content-Type")).toBe(
			"application/json",
		);
	});

	it("sends the JSON content type even without a body", async () => {
		mockApi({ "POST /api/auth/logout": { status: 204 } });
		await expect(
			api("/api/auth/logout", { method: "POST" }),
		).resolves.toBeUndefined();
		const init = vi.mocked(fetch).mock.calls[0][1] as RequestInit;
		expect(new Headers(init.headers).get("Content-Type")).toBe(
			"application/json",
		);
	});

	it("parses the error format into an ApiError with Retry-After", async () => {
		mockApi({
			"POST /api/auth/magic-link": apiError(429, "rate_limited", {
				"Retry-After": "42",
			}),
		});
		const error = await api("/api/auth/magic-link", {
			method: "POST",
			json: {},
		}).catch((e) => e);
		expect(error).toBeInstanceOf(ApiError);
		expect(error).toMatchObject({
			status: 429,
			code: "rate_limited",
			retryAfter: 42,
		});
	});

	it("handles error responses that aren't JSON", async () => {
		vi.stubGlobal(
			"fetch",
			vi.fn(async () => new Response("Bad Gateway", { status: 502 })),
		);
		const error = await api<never>("/api/health").catch((e: ApiError) => e);
		expect(error).toMatchObject({ status: 502, code: "error" });
		expect(error.retryAfter).toBeUndefined();
	});

	it("switches to signed out on a 401", async () => {
		const client = createQueryClient();
		client.setQueryData(meQueryKey, {
			id: "u1",
			email: "ann@example.org",
			name: null,
		});
		mockApi({ "PATCH /api/me": apiError(401, "unauthenticated") });

		await expect(
			api("/api/me", { method: "PATCH", json: { name: "x" } }),
		).rejects.toMatchObject({ status: 401 });
		expect(client.getQueryData(meQueryKey)).toBeNull();
	});
});
