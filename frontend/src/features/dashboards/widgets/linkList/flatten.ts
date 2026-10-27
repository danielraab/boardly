import type { FavouriteNode } from "#/features/favourites/api";

/** One link of a link list, as the server resolves it. */
export type ListLink = {
	id: string;
	title: string | null;
	url: string;
	iconImageId?: string;
};

/**
 * What a link list shows: its links, that its folder is gone (`true`), or that
 * its folder exists but its tag is gone (`"tag"`).
 */
export type LinkListData = { missing: true | "tag" } | { links: ListLink[] };

export type LinkListSort = "favourites" | "newest" | "alphabetical";

export const linkListSorts: LinkListSort[] = [
	"favourites",
	"newest",
	"alphabetical",
];

/** The link nodes in nodes, depth first, with subfolders when set. */
function linkNodes(
	nodes: FavouriteNode[],
	subfolders: boolean,
): FavouriteNode[] {
	return nodes.flatMap((n) => {
		if (n.kind === "link") return [n];
		return subfolders ? linkNodes(n.children ?? [], true) : [];
	});
}

const byID = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0);

/** What a link list shows for a link: its title, or its URL. */
const shownText = (n: FavouriteNode) => n.title || (n.url ?? "");

/**
 * The links in nodes as the server picks and orders them: depth first,
 * including those in subfolders when subfolders is set, only those with the
 * tag named tag when it isn't null, and sorted by sort. "newest" breaks ties
 * by ID descending; "alphabetical" ignores case and accents and breaks ties by
 * URL and ID. The server's collation may differ in rare cases.
 */
export function flattenLinks(
	nodes: FavouriteNode[],
	subfolders: boolean,
	tag: string | null = null,
	sort: LinkListSort = "favourites",
): ListLink[] {
	let links = linkNodes(nodes, subfolders);
	if (tag !== null) links = links.filter((n) => n.tags?.includes(tag));
	if (sort === "newest") {
		links.sort(
			(a, b) =>
				Date.parse(b.createdAt ?? "") - Date.parse(a.createdAt ?? "") ||
				byID(b.id, a.id),
		);
	} else if (sort === "alphabetical") {
		const collator = new Intl.Collator(undefined, { sensitivity: "base" });
		links.sort(
			(a, b) =>
				collator.compare(shownText(a), shownText(b)) ||
				byID(a.url ?? "", b.url ?? "") ||
				byID(a.id, b.id),
		);
	}
	return links.map((n) => ({
		id: n.id,
		title: n.title,
		url: n.url ?? "",
		iconImageId: n.iconImageId,
	}));
}
