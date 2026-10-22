import type { FavouriteNode } from "#/features/favourites/api";

/** One link of a link list, as the server resolves it. */
export type ListLink = {
	id: string;
	title: string | null;
	url: string;
	iconImageId?: string;
};

/** What a link list shows: its links, or that its folder is gone. */
export type LinkListData = { missing: true } | { links: ListLink[] };

/**
 * The links in nodes, depth first, including those in subfolders when
 * subfolders is set. The server flattens a folder in the same order.
 */
export function flattenLinks(
	nodes: FavouriteNode[],
	subfolders: boolean,
): ListLink[] {
	return nodes.flatMap((n): ListLink[] => {
		if (n.kind === "link") {
			return [
				{
					id: n.id,
					title: n.title,
					url: n.url ?? "",
					iconImageId: n.iconImageId,
				},
			];
		}
		return subfolders ? flattenLinks(n.children ?? [], true) : [];
	});
}
