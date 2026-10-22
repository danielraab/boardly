import { queryOptions } from "@tanstack/react-query";
import { api } from "#/lib/api";

/** A folder or link of the favourites tree. The root folder has no title. */
export type FavouriteNode = {
	id: string;
	kind: "folder" | "link";
	title: string | null;
	url?: string;
	tags?: string[];
	children?: FavouriteNode[];
	/** The favicon of a link's origin, when the server knows one. */
	iconImageId?: string;
};

export type Tag = { id: string; name: string };

export type DeleteMode = "delete" | "move";

export const favouritesQueryKey = ["favourites"] as const;

export const favouritesQueryOptions = queryOptions({
	queryKey: favouritesQueryKey,
	queryFn: () => api<FavouriteNode>("/api/favourites"),
});

export type NewFavourite =
	| { parentId: string; kind: "folder"; title: string }
	| {
			parentId: string;
			kind: "link";
			url: string;
			title: string;
			tags: string[];
	  };

export function createFavourite(node: NewFavourite) {
	return api<FavouriteNode>("/api/favourites", { method: "POST", json: node });
}

export function updateFavourite(
	id: string,
	patch: { title?: string; url?: string; tags?: string[] },
) {
	return api<FavouriteNode>(`/api/favourites/${encodeURIComponent(id)}`, {
		method: "PATCH",
		json: patch,
	});
}

export function moveFavourite(id: string, parentId: string, position: number) {
	return api<void>(`/api/favourites/${encodeURIComponent(id)}/move`, {
		method: "POST",
		json: { parentId, position },
	});
}

export function deleteFavourite(id: string, mode?: DeleteMode) {
	const query = mode ? `?children=${mode}` : "";
	return api<void>(`/api/favourites/${encodeURIComponent(id)}${query}`, {
		method: "DELETE",
	});
}

export function suggestTags(prefix: string) {
	return api<Tag[]>(`/api/tags?q=${encodeURIComponent(prefix)}`);
}
