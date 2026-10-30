import type { Note, NoteCategory } from "#/features/notes/api";
import type { NoteDoc } from "#/features/notes/document/types";
import {
	apiError,
	type MockResponse,
	mockApi,
	type Routes,
	signedIn,
} from "./api";

export const paragraphs = (...texts: string[]): NoteDoc => ({
	v: 1,
	blocks: texts.map((text) => ({
		type: "paragraph",
		content: [{ type: "text", text }],
	})),
});

let time = Date.UTC(2026, 9, 1);

export function note(id: string, fields: Partial<Note> = {}): Note {
	time += 60_000;
	return {
		id,
		title: id,
		icon: null,
		color: null,
		categoryId: null,
		content: paragraphs(`${id} text`),
		pinned: false,
		archived: false,
		version: 1,
		createdAt: new Date(time).toISOString(),
		updatedAt: new Date(time).toISOString(),
		attachments: [],
		...fields,
	};
}

export function category(
	id: string,
	title: string,
	parentId: string | null = null,
	fields: Partial<NoteCategory> = {},
): NoteCategory {
	return { id, title, parentId, icon: null, color: null, ...fields };
}

/**
 * A signed-in API with an in-memory notes server: it answers like the real
 * one (versions, ordering by last change, category deletes) and keeps its
 * state, so the page refetches what requests changed.
 */
export function fakeNotesServer(
	initialNotes: Note[] = [],
	initialCategories: NoteCategory[] = [],
	extra: Routes = {},
) {
	const notes = structuredClone(initialNotes);
	const categories = structuredClone(initialCategories);
	let next = 1;
	const touch = (n: Note) => {
		time += 60_000;
		n.updatedAt = new Date(time).toISOString();
		n.version += 1;
	};
	const sorted = () =>
		[...notes].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));

	const noteRoutes = (id: string): Routes => ({
		[`GET /api/notes/${id}`]: () => {
			const n = notes.find((x) => x.id === id);
			return n ? { body: n } : apiError(404, "not_found");
		},
		[`PATCH /api/notes/${id}`]: ({ body }) => {
			const n = notes.find((x) => x.id === id);
			if (!n) return apiError(404, "not_found");
			const { version, ...fields } = body as {
				version: number;
			} & Partial<Note>;
			if (version !== n.version) return apiError(409, "version_conflict");
			Object.assign(n, fields);
			touch(n);
			return { body: n };
		},
		[`DELETE /api/notes/${id}`]: () => {
			const i = notes.findIndex((x) => x.id === id);
			if (i < 0) return apiError(404, "not_found");
			notes.splice(i, 1);
			return { status: 204 };
		},
	});

	const categoryRoutes = (id: string): Routes => ({
		[`PATCH /api/note-categories/${id}`]: ({ body }) => {
			const c = categories.find((x) => x.id === id);
			if (!c) return apiError(404, "not_found");
			Object.assign(c, body);
			return { body: c };
		},
		[`DELETE /api/note-categories/${id}`]: () => {
			const i = categories.findIndex((x) => x.id === id);
			if (i < 0) return apiError(404, "not_found");
			const [gone] = categories.splice(i, 1);
			for (const c of categories)
				if (c.parentId === id) c.parentId = gone.parentId;
			for (const n of notes) if (n.categoryId === id) n.categoryId = null;
			return { status: 204 };
		},
	});

	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/notes": () => ({ body: sorted() }),
		"GET /api/note-categories": () => ({ body: categories }),
		"POST /api/notes": ({ body }): MockResponse => {
			const created = note(`new${next++}`, {
				title: null,
				content: { v: 1, blocks: [] },
				...(body as Partial<Note>),
				version: 1,
			});
			notes.push(created);
			api.set(noteRoutes(created.id));
			return { status: 201, body: created };
		},
		"POST /api/note-categories": ({ body }): MockResponse => {
			const created = { ...category(`c${next++}`, ""), ...(body as object) };
			categories.push(created);
			api.set(categoryRoutes(created.id));
			return { status: 201, body: created };
		},
		...Object.assign({}, ...notes.map((n) => noteRoutes(n.id))),
		...Object.assign({}, ...categories.map((c) => categoryRoutes(c.id))),
		...extra,
	});
	return { api, notes, categories };
}
