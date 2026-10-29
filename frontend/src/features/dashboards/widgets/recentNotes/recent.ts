import type { Note, NoteCategory, NoteIcon } from "#/features/notes/api";
import { categoryPath } from "#/features/notes/categories/tree";
import type { NoteColour } from "#/features/notes/colours";
import { displayTitle } from "#/features/notes/document/text";

/** Which time of a note a recent notes widget sorts and limits by. */
export type RecentNotesSort = "created" | "updated";

/** One note of a recent notes widget, as the server resolves it; id and path are the owner's. */
export type RecentNote = {
	id?: string;
	/** The display title, or null for "Untitled note". */
	title: string | null;
	icon: NoteIcon | null;
	color: NoteColour | null;
	/** The time sorted by, as RFC 3339. */
	time: string;
	category: { title: string; path?: string[] } | null;
};

export type RecentNotesData = { notes: RecentNote[] };

const dayMs = 24 * 60 * 60 * 1000;

/**
 * The newest of notes, as the server picks them: by creation time or time of
 * last change, within the last days × 24 hours (any time without days),
 * newest first, then by ID descending, at most limit.
 */
export function recentNotes(
	notes: Note[],
	categories: NoteCategory[],
	sortBy: RecentNotesSort,
	limit: number,
	days: number | null,
	now: Date,
): RecentNote[] {
	const since = days === null ? -Infinity : now.getTime() - days * dayMs;
	const timeOf = (n: Note) =>
		sortBy === "updated" ? n.updatedAt : n.createdAt;
	return notes
		.filter((n) => Date.parse(timeOf(n)) >= since)
		.sort(
			(a, b) =>
				Date.parse(timeOf(b)) - Date.parse(timeOf(a)) ||
				(a.id < b.id ? 1 : a.id > b.id ? -1 : 0),
		)
		.slice(0, limit)
		.map((n) => {
			const category =
				n.categoryId && categories.find((c) => c.id === n.categoryId);
			return {
				id: n.id,
				title: displayTitle(n),
				icon: n.icon,
				color: n.color,
				time: timeOf(n),
				category: category
					? {
							title: category.title,
							path: categoryPath(categories, category.id),
						}
					: null,
			};
		});
}
