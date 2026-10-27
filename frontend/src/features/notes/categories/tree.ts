import type { Note, NoteCategory } from "../api";

/** A category with its subcategories, sorted by title. */
export type CategoryTreeNode = {
	category: NoteCategory;
	children: CategoryTreeNode[];
	depth: number;
};

/**
 * The categories as a tree, siblings ordered by title, case-insensitively in
 * the given language. A category whose parent is missing is shown at the top
 * level.
 */
export function buildTree(
	categories: NoteCategory[],
	locale: string,
): CategoryTreeNode[] {
	const ids = new Set(categories.map((c) => c.id));
	const byParent = new Map<string | null, NoteCategory[]>();
	for (const c of categories) {
		const parent = c.parentId && ids.has(c.parentId) ? c.parentId : null;
		byParent.set(parent, [...(byParent.get(parent) ?? []), c]);
	}
	const collator = new Intl.Collator(locale, { sensitivity: "base" });
	const build = (parent: string | null, depth: number): CategoryTreeNode[] =>
		(byParent.get(parent) ?? [])
			.sort(
				(a, b) =>
					collator.compare(a.title, b.title) || a.id.localeCompare(b.id),
			)
			.map((category) => ({
				category,
				children: build(category.id, depth + 1),
				depth,
			}));
	return build(null, 0);
}

/** The tree's nodes in display order, parents before their children. */
export function flatten(tree: CategoryTreeNode[]): CategoryTreeNode[] {
	return tree.flatMap((node) => [node, ...flatten(node.children)]);
}

/** The IDs of a category and all its subcategories at every depth. */
export function subtreeIds(
	categories: NoteCategory[],
	id: string,
): Set<string> {
	const ids = new Set([id]);
	let grew = true;
	while (grew) {
		grew = false;
		for (const c of categories) {
			if (c.parentId && ids.has(c.parentId) && !ids.has(c.id)) {
				ids.add(c.id);
				grew = true;
			}
		}
	}
	return ids;
}

/** The titles from the top-level category down to id, e.g. ["Home", "Garden"]. */
export function categoryPath(categories: NoteCategory[], id: string): string[] {
	const byId = new Map(categories.map((c) => [c.id, c]));
	const path: string[] = [];
	const seen = new Set<string>();
	let current = byId.get(id);
	while (current && !seen.has(current.id)) {
		seen.add(current.id);
		path.unshift(current.title);
		current = current.parentId ? byId.get(current.parentId) : undefined;
	}
	return path;
}

/** The side panel's choice: all notes, the uncategorized ones, or a category. */
export type CategoryFilter = "all" | "none" | { id: string };

/** The notes a filter shows, in the order given. */
export function filterNotes(
	notes: Note[],
	categories: NoteCategory[],
	filter: CategoryFilter,
): Note[] {
	if (filter === "all") return notes;
	if (filter === "none") return notes.filter((n) => n.categoryId === null);
	const ids = subtreeIds(categories, filter.id);
	return notes.filter((n) => n.categoryId !== null && ids.has(n.categoryId));
}
