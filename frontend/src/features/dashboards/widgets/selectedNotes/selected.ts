import type { Note, NoteCategory } from "#/features/notes/api";
import { categoryPath } from "#/features/notes/categories/tree";
import { displayTitle } from "#/features/notes/document/text";
import type { NoteListData } from "../noteList/list";

/** At most this many notes can be selected, as on the server. */
export const maxSelectedNotes = 100;

/**
 * The notes selected notes show, as the server resolves them: those of
 * noteIds that exist, in that order, each with its category if it has one.
 * When none exists, they are missing.
 */
export function selectedNotes(
	notes: Note[],
	categories: NoteCategory[],
	config: { noteIds: string[] },
): NoteListData {
	const byID = new Map(notes.map((n) => [n.id, n]));
	const shown = config.noteIds.flatMap((id) => byID.get(id) ?? []);
	if (shown.length === 0) return { missing: true };
	return {
		notes: shown.map((note) => {
			const category =
				note.categoryId !== null
					? categories.find((c) => c.id === note.categoryId)
					: undefined;
			return {
				id: note.id,
				title: displayTitle(note),
				icon: note.icon,
				color: note.color,
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

/** A note the search offers, with its display title and category path. */
export type NoteChoice = {
	note: Note;
	title: string | null;
	path: string[];
};

// Lower case without accents, so "arzte" finds "Ärzte".
const fold = (s: string) =>
	s
		.normalize("NFD")
		.replace(/\p{Mn}/gu, "")
		.toLowerCase();

/**
 * The notes the search offers for query: those not in chosen whose display
 * title or category path contains it, ignoring case and accents, sorted by
 * display title with untitled notes last.
 */
export function noteChoices(
	notes: Note[],
	categories: NoteCategory[],
	chosen: string[],
	query: string,
): NoteChoice[] {
	const taken = new Set(chosen);
	const q = fold(query.trim());
	const collator = new Intl.Collator(undefined, { sensitivity: "base" });
	return notes
		.filter((n) => !taken.has(n.id))
		.map((note) => ({
			note,
			title: displayTitle(note),
			path:
				note.categoryId !== null
					? categoryPath(categories, note.categoryId)
					: [],
		}))
		.filter(
			(c) =>
				q === "" ||
				fold(c.title ?? "").includes(q) ||
				fold(c.path.join(" › ")).includes(q),
		)
		.sort((a, b) => {
			if (a.title === null || b.title === null)
				return (a.title === null ? 1 : 0) - (b.title === null ? 1 : 0);
			return collator.compare(a.title, b.title);
		});
}
