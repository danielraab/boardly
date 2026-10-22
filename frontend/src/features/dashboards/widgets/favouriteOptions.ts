import type { FavouriteNode } from "#/features/favourites/api";
import { label } from "#/features/favourites/tree";

export type FavouriteOption = { node: FavouriteNode; path: string };

/** Every folder with its path from the root ("Favourites / Work"), in tree order. */
export function folderOptions(
	root: FavouriteNode,
	rootName: string,
): FavouriteOption[] {
	const visit = (node: FavouriteNode, path: string): FavouriteOption[] => [
		{ node, path },
		...(node.children ?? [])
			.filter((c) => c.kind === "folder")
			.flatMap((c) => visit(c, `${path} / ${label(c)}`)),
	];
	return visit(root, rootName);
}

/** Every link with the path of its folder ("Work / Mail"), in tree order. */
export function linkOptions(root: FavouriteNode): FavouriteOption[] {
	const visit = (node: FavouriteNode, path: string): FavouriteOption[] =>
		(node.children ?? []).flatMap((c) => {
			const own = path ? `${path} / ${label(c)}` : label(c);
			return c.kind === "folder" ? visit(c, own) : [{ node: c, path: own }];
		});
	return visit(root, "");
}
