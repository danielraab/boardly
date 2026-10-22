import { useQuery } from "@tanstack/react-query";
import { ApiError } from "#/lib/api";
import { faviconFor } from "./api";

/** The origin of an http(s) URL, or null. */
export function originOf(url: string | null): string | null {
	if (!url) return null;
	try {
		const { origin, protocol } = new URL(url);
		return protocol === "http:" || protocol === "https:" ? origin : null;
	} catch {
		return null;
	}
}

/**
 * The website icon of a URL: knownId when the caller has it (a favourite's
 * iconImageId), otherwise the server's favicon of the URL's site, looked up
 * once per site. No favicon and a blocked address both mean null.
 */
export function useFavicon(url: string | null, knownId?: string | null) {
	const origin = originOf(url);
	const query = useQuery({
		queryKey: ["favicon", origin],
		queryFn: async () => {
			try {
				return await faviconFor(url ?? "");
			} catch (err) {
				if (err instanceof ApiError && err.status < 500) return null;
				throw err;
			}
		},
		enabled: origin !== null && !knownId,
		staleTime: Number.POSITIVE_INFINITY,
		retry: false,
	});
	if (knownId) return { id: knownId, loading: false };
	return {
		id: query.data ?? null,
		loading: origin !== null && query.isPending,
	};
}
