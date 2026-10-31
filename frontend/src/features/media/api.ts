import { queryOptions } from "@tanstack/react-query";
import type { NoteIcon } from "#/features/notes/api";
import { api } from "#/lib/api";

/** What the player needs of a media source; visitors get only this. */
export type PlayableSource = {
	id: string;
	kind: "stream";
	title: string;
	url: string;
	icon: NoteIcon | null;
};

/** A media source as the media API returns it. */
export type MediaSource = PlayableSource & { position: number };

export const mediaQueryKey = ["media"] as const;

const sourcePath = (id: string) => `/api/media/${encodeURIComponent(id)}`;

export const mediaQueryOptions = queryOptions({
	queryKey: mediaQueryKey,
	queryFn: () => api<MediaSource[]>("/api/media"),
});

export type NewStation = {
	title: string;
	url: string;
	/** A logo the server fetches as the station's icon. */
	logoUrl?: string;
};

/**
 * Adds a stream at the end of the library. The server checks the URL first;
 * a failed check throws an ApiError with invalid_url, stream_not_https or
 * stream_unreachable.
 */
export function createStation(station: NewStation) {
	return api<MediaSource>("/api/media", {
		method: "POST",
		json: { kind: "stream", ...station },
	});
}

export type SourcePatch = {
	title?: string;
	url?: string;
	icon?: NoteIcon | null;
};

export function updateSource(id: string, patch: SourcePatch) {
	return api<MediaSource>(sourcePath(id), { method: "PATCH", json: patch });
}

/** Sets the order of all of the user's sources. */
export function reorderSources(ids: string[]) {
	return api<void>("/api/media/order", { method: "PUT", json: ids });
}

export function deleteSource(id: string) {
	return api<void>(sourcePath(id), { method: "DELETE" });
}

/** The URL's host, for showing where a station streams from. */
export function hostOf(url: string) {
	try {
		return new URL(url).host;
	} catch {
		return url;
	}
}

/** A stream URL in a form that compares equal across small differences. */
export function normalizeStreamUrl(url: string) {
	try {
		const u = new URL(url);
		return `${u.protocol}//${u.host.toLowerCase()}${u.pathname.replace(/\/+$/, "")}${u.search}`;
	} catch {
		return url.trim();
	}
}
