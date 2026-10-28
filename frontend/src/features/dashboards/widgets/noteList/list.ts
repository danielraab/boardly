import type { Note, NoteCategory, NoteIcon } from "#/features/notes/api";
import { categoryPath, subtreeIds } from "#/features/notes/categories/tree";
import { displayTitle } from "#/features/notes/document/text";

/** Which notes a note list shows: all, the uncategorized ones, or a category's. */
export type NoteListCategory = "all" | "none" | "category";

/** The order of a note list. */
export type NoteListSort = "updated" | "created" | "title";

/** One note of a note list, as the server resolves it; id and path are the owner's. */
export type ListedNote = {
	id?: string;
	/** The display title, or null for "Untitled note". */
	title: string | null;
	icon: NoteIcon | null;
	/** The category of a note that isn't directly in the chosen one. */
	category: { title: string; path?: string[] } | null;
};

/** What a note list shows: its notes, or that its category is gone. */
export type NoteListData = { notes: ListedNote[] } | { missing: true };

const byID = (a: string, b: string) => (a < b ? -1 : a > b ? 1 : 0);

/**
 * The notes a note list shows, as the server picks and orders them: "updated"
 * and "created" newest first, then by ID descending; "title" by display title
 * ignoring case and accents, untitled last, then by ID; at most limit. The
 * server's collation may differ in rare cases.
 */
export function noteList(
	notes: Note[],
	categories: NoteCategory[],
	config: {
		category: NoteListCategory;
		categoryId: string | null;
		subcategories: boolean;
		sort: NoteListSort;
		limit: number;
	},
): NoteListData {
	const chosen = config.category === "category" ? config.categoryId : null;
	let shown: Note[];
	if (config.category === "all") shown = notes;
	else if (config.category === "none")
		shown = notes.filter((n) => n.categoryId === null);
	else {
		if (!chosen || !categories.some((c) => c.id === chosen))
			return { missing: true };
		const ids = config.subcategories
			? subtreeIds(categories, chosen)
			: new Set([chosen]);
		shown = notes.filter((n) => n.categoryId !== null && ids.has(n.categoryId));
	}

	const titled = shown.map((n) => ({ note: n, title: displayTitle(n) }));
	if (config.sort === "title") {
		const collator = new Intl.Collator(undefined, { sensitivity: "base" });
		titled.sort((a, b) => {
			if (a.title === null || b.title === null)
				return (
					(a.title === null ? 1 : 0) - (b.title === null ? 1 : 0) ||
					byID(a.note.id, b.note.id)
				);
			return collator.compare(a.title, b.title) || byID(a.note.id, b.note.id);
		});
	} else {
		const timeOf = (n: Note) =>
			config.sort === "created" ? n.createdAt : n.updatedAt;
		titled.sort(
			(a, b) =>
				Date.parse(timeOf(b.note)) - Date.parse(timeOf(a.note)) ||
				byID(b.note.id, a.note.id),
		);
	}

	return {
		notes: titled.slice(0, config.limit).map(({ note, title }) => {
			const category =
				note.categoryId !== null && note.categoryId !== chosen
					? categories.find((c) => c.id === note.categoryId)
					: undefined;
			return {
				id: note.id,
				title,
				icon: note.icon,
				category: category
					? {
							title: category.title,
							path: categoryPath(categories, category.id),
						}
					: null,
			};
		}),
	};
}
