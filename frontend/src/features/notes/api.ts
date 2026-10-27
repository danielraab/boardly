import { queryOptions } from "@tanstack/react-query";
import { ApiError, api } from "#/lib/api";
import type { NoteColour } from "./colours";
import type { NoteDoc } from "./document/types";

/** The icon of a note or a category. */
export type NoteIcon =
	| { type: "symbol"; name: string }
	| { type: "image"; imageId: string };

export type Note = {
	id: string;
	title: string | null;
	icon: NoteIcon | null;
	color: NoteColour | null;
	categoryId: string | null;
	content: NoteDoc;
	version: number;
	createdAt: string;
	updatedAt: string;
};

/** The fields of a note that can be changed. */
export type NoteFields = Pick<
	Note,
	"title" | "icon" | "color" | "categoryId" | "content"
>;

export type NoteCategory = {
	id: string;
	title: string;
	icon: NoteIcon | null;
	color: NoteColour | null;
	parentId: string | null;
};

export type CategoryFields = Omit<NoteCategory, "id">;

export const notesQueryKey = ["notes"] as const;
export const noteCategoriesQueryKey = ["note-categories"] as const;

const notePath = (id: string) => `/api/notes/${encodeURIComponent(id)}`;
const categoryPath = (id: string) =>
	`/api/note-categories/${encodeURIComponent(id)}`;

export const notesQueryOptions = queryOptions({
	queryKey: notesQueryKey,
	queryFn: () => api<Note[]>("/api/notes"),
});

export const noteQueryOptions = (id: string) =>
	queryOptions({
		queryKey: [...notesQueryKey, id],
		queryFn: () => api<Note>(notePath(id)),
	});

export const noteCategoriesQueryOptions = queryOptions({
	queryKey: noteCategoriesQueryKey,
	queryFn: () => api<NoteCategory[]>("/api/note-categories"),
});

export function createNote(fields: Partial<NoteFields>) {
	return api<Note>("/api/notes", { method: "POST", json: fields });
}

/**
 * Changes a note that is at version. A note changed elsewhere in the
 * meantime throws an ApiError with the code version_conflict.
 */
export function updateNote(
	id: string,
	version: number,
	fields: Partial<NoteFields>,
	init: { keepalive?: boolean } = {},
) {
	return api<Note>(notePath(id), {
		method: "PATCH",
		json: { version, ...fields },
		...init,
	});
}

export function deleteNote(id: string) {
	return api<void>(notePath(id), { method: "DELETE" });
}

export function createCategory(fields: CategoryFields) {
	return api<NoteCategory>("/api/note-categories", {
		method: "POST",
		json: fields,
	});
}

export function updateCategory(id: string, fields: Partial<CategoryFields>) {
	return api<NoteCategory>(categoryPath(id), { method: "PATCH", json: fields });
}

export function deleteCategory(id: string) {
	return api<void>(categoryPath(id), { method: "DELETE" });
}

export function isVersionConflict(err: unknown): boolean {
	return err instanceof ApiError && err.code === "version_conflict";
}
