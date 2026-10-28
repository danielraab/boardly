import type { FavouriteNode } from "#/features/favourites/api";
import type { ListLink } from "../linkList/flatten";

/** One link of a recent favourites widget, as the server resolves it. */
export type RecentLink = ListLink & {
	/** When the link was added, as RFC 3339. */
	createdAt: string;
};

/** What the widget shows: its links, or that its folder is gone. */
export type RecentData = { missing: true } | { links: RecentLink[] };

export const maxRecentLimit = 100;
export const maxRecentDays = 365;

/** The whole number in value if it is from 1 to max, otherwise null. */
export function parseCount(value: string, max: number): number | null {
	const n = Number(value);
	return value.trim() !== "" && Number.isInteger(n) && n >= 1 && n <= max
		? n
		: null;
}

const dayMs = 24 * 60 * 60 * 1000;

/**
 * The newest links in nodes and their subfolders, as the server picks them:
 * added within the last days × 24 hours (any time without days), newest
 * first, then by ID descending, at most limit.
 */
export function recentLinks(
	nodes: FavouriteNode[],
	limit: number,
	days: number | null,
	now: Date,
): RecentLink[] {
	const since = days === null ? -Infinity : now.getTime() - days * dayMs;
	const links: RecentLink[] = [];
	const walk = (list: FavouriteNode[]) => {
		for (const n of list) {
			if (n.kind === "folder") {
				walk(n.children ?? []);
			} else if (n.createdAt && Date.parse(n.createdAt) >= since) {
				links.push({
					id: n.id,
					title: n.title,
					url: n.url ?? "",
					iconImageId: n.iconImageId,
					createdAt: n.createdAt,
				});
			}
		}
	};
	walk(nodes);
	links.sort(
		(a, b) =>
			Date.parse(b.createdAt) - Date.parse(a.createdAt) ||
			(a.id < b.id ? 1 : a.id > b.id ? -1 : 0),
	);
	return links.slice(0, limit);
}
