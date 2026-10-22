import { queryOptions } from "@tanstack/react-query";
import { api } from "#/lib/api";

export type AppTokenApp = "karakeep" | "linkwarden";

/** The apps' product names, which aren't translated. */
export const appNames: Record<AppTokenApp, string> = {
	karakeep: "Karakeep",
	linkwarden: "Linkwarden",
};

export type AppTokenStatus = "active" | "expired" | "revoked";

/** An app token as the server lists it; the secret is never part of it. */
export type AppToken = {
	id: string;
	name: string;
	app: AppTokenApp;
	/** The first 12 characters of the secret. */
	prefix: string;
	/** null once the root folder was deleted. */
	rootId: string | null;
	createdAt: string;
	lastUsedAt: string | null;
	expiresAt: string | null;
	revokedAt: string | null;
	status: AppTokenStatus;
};

/** A token just created, with its secret, which is only returned this once. */
export type CreatedAppToken = AppToken & { secret: string };

export type NewAppToken = {
	name: string;
	app: AppTokenApp;
	rootId: string;
	/** RFC 3339, or null for no expiry. */
	expiresAt: string | null;
};

export const appTokensQueryKey = ["app-tokens"] as const;

export const appTokensQueryOptions = queryOptions({
	queryKey: appTokensQueryKey,
	queryFn: () => api<AppToken[]>("/api/app-tokens"),
});

export function createAppToken(token: NewAppToken) {
	return api<CreatedAppToken>("/api/app-tokens", {
		method: "POST",
		json: token,
	});
}

export function revokeAppToken(id: string) {
	return api<AppToken>(`/api/app-tokens/${encodeURIComponent(id)}/revoke`, {
		method: "POST",
	});
}

export function deleteAppToken(id: string) {
	return api<void>(`/api/app-tokens/${encodeURIComponent(id)}`, {
		method: "DELETE",
	});
}

/** The address to enter in the app, which calls `<address>/api/v1/…`. */
export function serverAddress(app: AppTokenApp) {
	return `${window.location.origin}/api/${app}`;
}
