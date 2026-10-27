import { describe, expect, it } from "vitest";
import { apiError, mockApi } from "#/test/api";
import {
	createCategory,
	createNote,
	deleteCategory,
	deleteNote,
	isVersionConflict,
	updateCategory,
	updateNote,
} from "./api";

describe("notes API", () => {
	it("sends notes and categories to their endpoints", async () => {
		const { calls } = mockApi({
			"POST /api/notes": { status: 201, body: { id: "n1" } },
			"PATCH /api/notes/n1": { body: { id: "n1" } },
			"DELETE /api/notes/n1": { status: 204 },
			"POST /api/note-categories": { status: 201, body: { id: "c1" } },
			"PATCH /api/note-categories/c1": { body: { id: "c1" } },
			"DELETE /api/note-categories/c1": { status: 204 },
		});
		await createNote({ title: "Groceries" });
		await updateNote("n1", 3, { color: "green" });
		await deleteNote("n1");
		await createCategory({
			title: "Home",
			icon: null,
			color: null,
			parentId: null,
		});
		await updateCategory("c1", { parentId: null });
		await deleteCategory("c1");
		expect(calls.map((c) => [c.method, c.path, c.body])).toEqual([
			["POST", "/api/notes", { title: "Groceries" }],
			["PATCH", "/api/notes/n1", { version: 3, color: "green" }],
			["DELETE", "/api/notes/n1", undefined],
			[
				"POST",
				"/api/note-categories",
				{ title: "Home", icon: null, color: null, parentId: null },
			],
			["PATCH", "/api/note-categories/c1", { parentId: null }],
			["DELETE", "/api/note-categories/c1", undefined],
		]);
	});

	it("recognizes version conflicts", async () => {
		mockApi({ "PATCH /api/notes/n1": apiError(409, "version_conflict") });
		const err = await updateNote("n1", 1, { title: "x" }).catch((e) => e);
		expect(isVersionConflict(err)).toBe(true);
		expect(isVersionConflict(new Error("x"))).toBe(false);
	});
});
