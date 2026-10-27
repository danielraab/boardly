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

const units: [Intl.RelativeTimeFormatUnit, number][] = [
	["year", 365 * dayMs],
	["month", 30 * dayMs],
	["day", dayMs],
	["hour", 60 * 60 * 1000],
	["minute", 60 * 1000],
];

/**
 * How long ago date was, in the largest unit that fits, like "3 days ago" or
 * "yesterday"; "now" under a minute.
 */
export function relativeTime(date: Date, now: Date, language: string): string {
	const format = new Intl.RelativeTimeFormat(language, { numeric: "auto" });
	const ago = Math.max(0, now.getTime() - date.getTime());
	for (const [unit, ms] of units) {
		if (ago >= ms) return format.format(-Math.floor(ago / ms), unit);
	}
	return format.format(0, "second");
}

/** The full date and time of date, for a tooltip. */
export function fullDate(date: Date, language: string): string {
	return new Intl.DateTimeFormat(language, {
		dateStyle: "medium",
		timeStyle: "short",
	}).format(date);
}
