import {
	QueryClient,
	queryOptions,
	useQuery,
	useQueryClient,
} from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { ApiError, api, setUnauthorizedHandler } from "#/lib/api";

export type User = {
	id: string;
	email: string;
	name: string | null;
	/** First part of the user's dashboard addresses: /<slug>/<dashboard-slug>. */
	slug: string;
	/** Default folder of the "Save link" page, or null for the root. */
	shareFolderId: string | null;
	noteCategoryId: string | null;
	/** End of the passkey enrollment window (RFC 3339), or null when none is open. */
	passkeyEnrollmentUntil: string | null;
};

export const meQueryKey = ["me"] as const;

/** The signed-in user, or null when signed out. */
export const meQueryOptions = queryOptions({
	queryKey: meQueryKey,
	queryFn: async (): Promise<User | null> => {
		try {
			return await api<User>("/api/me");
		} catch (e) {
			if (e instanceof ApiError && e.status === 401) return null;
			throw e;
		}
	},
	retry: false,
	staleTime: 5 * 60_000,
});

export function useMe() {
	return useQuery(meQueryOptions);
}

/** Creates the app's query client; a 401 from any request switches it to signed out. */
export function createQueryClient() {
	const client = new QueryClient({
		defaultOptions: { queries: { retry: false } },
	});
	setUnauthorizedHandler(() => client.setQueryData(meQueryKey, null));
	return client;
}

/** Drops all cached user data and marks the visitor as signed out. */
export function resetToSignedOut(client: QueryClient) {
	client.removeQueries({
		predicate: (q) => q.queryKey[0] !== meQueryKey[0],
	});
	client.setQueryData(meQueryKey, null);
}

export function useSignOut() {
	const client = useQueryClient();
	const navigate = useNavigate();
	return async () => {
		try {
			await api("/api/auth/logout", { method: "POST" });
		} finally {
			resetToSignedOut(client);
			await navigate({ to: "/" });
		}
	};
}
