import { api } from "#/lib/api";

/** The ID of the favicon of a URL's site; throws ApiError when there is none. */
export async function faviconFor(url: string) {
	const { id } = await api<{ id: string }>("/api/favicons", {
		method: "POST",
		json: { url },
	});
	return id;
}
