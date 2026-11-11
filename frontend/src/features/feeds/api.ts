import { infiniteQueryOptions, queryOptions } from "@tanstack/react-query";
import type { PlayableSource } from "#/features/media/api";
import type { NoteIcon } from "#/features/notes/api";
import { api } from "#/lib/api";

export type FeedView = "latest" | "read_later" | "liked";

export const feedViews: FeedView[] = ["latest", "read_later", "liked"];

/** A user's subscription to a feed. */
export type FeedSubscription = {
	id: string;
	feedId: string;
	url: string;
	title: string;
	customTitle: string | null;
	siteUrl: string | null;
	icon: NoteIcon | null;
	faviconId: string | null;
	position: number;
	lastFetchedAt: string | null;
	failing: boolean;
};

export type FeedAudio = {
	url: string;
	type: string | null;
	durationSeconds: number | null;
};

/** The feed an item is shown with. Visitors get no subscriptionId. */
export type FeedItemFeed = {
	title: string;
	subscriptionId?: string | null;
	icon: NoteIcon | null;
	faviconId: string | null;
};

/** A feed item. Visitors of a public dashboard get no flags. */
export type FeedItem = {
	id: string;
	title: string;
	link: string | null;
	summary: string | null;
	publishedAt: string;
	audio: FeedAudio | null;
	feed: FeedItemFeed;
	readLaterAt?: string | null;
	likedAt?: string | null;
};

export type FeedItemsPage = { items: FeedItem[]; nextCursor: string | null };

export const feedsQueryKey = ["feeds"] as const;
export const feedItemsQueryKey = ["feedItems"] as const;

export const feedsQueryOptions = queryOptions({
	queryKey: feedsQueryKey,
	queryFn: () => api<FeedSubscription[]>("/api/feeds"),
});

const pageSize = 50;

/** A search and a feed filter on top of a view; empty means none. */
export type FeedItemsFilter = { q?: string; feedId?: string };

function itemsPath(
	view: FeedView,
	limit: number,
	cursor?: string,
	{ q, feedId }: FeedItemsFilter = {},
) {
	const query = new URLSearchParams({ view, limit: String(limit) });
	if (feedId) query.set("feedId", feedId);
	if (q?.trim()) query.set("q", q.trim());
	if (cursor) query.set("cursor", cursor);
	return `/api/feed-items?${query}`;
}

/** The items of a view, page by page, optionally searched and filtered. */
export const feedItemsQueryOptions = (
	view: FeedView,
	filter: FeedItemsFilter = {},
) => {
	const q = filter.q?.trim() ?? "";
	const feedId = filter.feedId ?? "";
	return infiniteQueryOptions({
		queryKey: [...feedItemsQueryKey, view, { q, feedId }],
		queryFn: ({ pageParam }) =>
			api<FeedItemsPage>(itemsPath(view, pageSize, pageParam, { q, feedId })),
		initialPageParam: undefined as string | undefined,
		getNextPageParam: (last) => last.nextCursor ?? undefined,
	});
};

/** One page of a view, for previewing a widget in the editor. */
export const feedPreviewQueryOptions = (view: FeedView) =>
	queryOptions({
		queryKey: [...feedItemsQueryKey, view, "preview"],
		queryFn: () => api<FeedItemsPage>(itemsPath(view, 100)),
	});

const feedPath = (id: string) => `/api/feeds/${encodeURIComponent(id)}`;

/**
 * Subscribes to a feed, or the one a page advertises. A failed check throws
 * an ApiError with invalid_url, feed_unreachable, feed_not_found,
 * already_subscribed or rate_limited.
 */
export function createFeed(url: string) {
	return api<FeedSubscription>("/api/feeds", {
		method: "POST",
		json: { url },
	});
}

export type FeedPatch = {
	customTitle?: string | null;
	icon?: NoteIcon | null;
};

export function updateFeed(id: string, patch: FeedPatch) {
	return api<FeedSubscription>(feedPath(id), { method: "PATCH", json: patch });
}

/** Sets the order of all of the user's subscriptions. */
export function reorderFeeds(ids: string[]) {
	return api<void>("/api/feeds/order", { method: "PUT", json: ids });
}

export function deleteFeed(id: string) {
	return api<void>(feedPath(id), { method: "DELETE" });
}

export type FlagPatch = { readLater?: boolean; liked?: boolean };

export function setFeedItemFlags(id: string, patch: FlagPatch) {
	return api<FeedItem>(`/api/feed-items/${encodeURIComponent(id)}`, {
		method: "PATCH",
		json: patch,
	});
}

/** A feed's chosen icon, or else the shared favicon of its site. */
export function feedIcon(feed: {
	icon: NoteIcon | null;
	faviconId: string | null;
}): NoteIcon | null {
	if (feed.icon) return feed.icon;
	return feed.faviconId ? { type: "image", imageId: feed.faviconId } : null;
}

/** What the player needs to play an item that has audio. */
export function episodeSource(
	item: FeedItem & { audio: FeedAudio },
): PlayableSource {
	return {
		id: item.id,
		kind: "episode",
		title: item.title,
		url: item.audio.url,
		icon: feedIcon(item.feed),
		subtitle: item.feed.title,
		durationSeconds: item.audio.durationSeconds ?? undefined,
	};
}
