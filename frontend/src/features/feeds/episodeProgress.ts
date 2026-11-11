import type { QueryClient } from "@tanstack/react-query";
import { dashboardViewQueryKey } from "#/features/dashboards/api";
import {
	type FeedItem,
	type FeedListen,
	feedItemsQueryKey,
	type ListenPatch,
	saveListen,
} from "./api";

/** The key of positions that earlier versions kept in the browser. */
export const oldPositionsKey = "boardly.player.positions";

/**
 * Returns data with every feed item of this ID given the listen record,
 * sharing everything else. Works for item pages and dashboard views alike.
 */
function withListen(data: unknown, id: string, listen: FeedListen | null) {
	const visit = (value: unknown): unknown => {
		if (Array.isArray(value)) {
			let changed = false;
			const out = value.map((v) => {
				const next = visit(v);
				if (next !== v) changed = true;
				return next;
			});
			return changed ? out : value;
		}
		if (value === null || typeof value !== "object") return value;
		const record = value as Record<string, unknown>;
		// Only the owner's items carry a listen record to change.
		if (record.id === id && "audio" in record && "listen" in record)
			return { ...record, listen };
		let changed = false;
		const out: Record<string, unknown> = {};
		for (const [key, v] of Object.entries(record)) {
			const next = visit(v);
			if (next !== v) changed = true;
			out[key] = next;
		}
		return changed ? out : value;
	};
	return visit(data);
}

/**
 * Shows an item's new listen record everywhere it is cached: in the feeds
 * page's lists and in feed widgets. A finished, unplayed or removed record
 * can also move items between lists, so those are fetched again.
 */
export function showListen(
	queryClient: QueryClient,
	id: string,
	listen: FeedListen | null,
	{ refetch = false } = {},
) {
	for (const queryKey of [feedItemsQueryKey, dashboardViewQueryKey]) {
		queryClient.setQueriesData({ queryKey }, (data: unknown) =>
			data === undefined ? data : withListen(data, id, listen),
		);
		if (refetch) void queryClient.invalidateQueries({ queryKey });
	}
}

// The last request per episode, so that requests reach the server in the
// order they were made: a menu action after the player's last save wins.
const queues = new Map<string, Promise<unknown>>();

/** Runs task after the earlier listen requests of the episode. */
export function inOrder<T>(id: string, task: () => Promise<T>): Promise<T> {
	const run = (queues.get(id) ?? Promise.resolve()).then(task, task);
	const tail = run.catch(() => {});
	queues.set(id, tail);
	void tail.then(() => {
		if (queues.get(id) === tail) queues.delete(id);
	});
	return run;
}

/**
 * Saves the position in an episode on the server and shows it in cached
 * rows. A failed save is dropped: the next one carries a newer position.
 */
export async function saveProgress(
	queryClient: QueryClient,
	id: string,
	patch: ListenPatch,
	{ keepalive = false } = {},
) {
	let item: FeedItem;
	try {
		// A save as the page goes away can't wait for earlier ones.
		item = keepalive
			? await saveListen(id, patch, { keepalive })
			: await inOrder(id, () => saveListen(id, patch));
	} catch {
		return;
	}
	showListen(queryClient, id, item.listen ?? null, {
		refetch: Boolean(patch.finished),
	});
}
