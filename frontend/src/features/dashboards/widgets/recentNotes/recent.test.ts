import { describe, expect, it } from "vitest";
import type { Note, NoteCategory } from "#/features/notes/api";
import { recentNotes } from "./recent";

const now = new Date("2026-10-27T12:00:00Z");
const day = 24 * 60 * 60 * 1000;
const daysAgo = (n: number) => new Date(now.getTime() - n * day).toISOString();

const note = (
	id: string,
	created: number,
	updated: number,
	fields: Partial<Note> = {},
): Note => ({
	id,
	title: id,
	icon: null,
	color: null,
	categoryId: null,
	content: { v: 1, blocks: [] },
	pinned: false,
	archived: false,
	version: 1,
	createdAt: daysAgo(created),
	updatedAt: daysAgo(updated),
	...fields,
});

const categories: NoteCategory[] = [
	{ id: "home", title: "Home", icon: null, color: null, parentId: null },
	{ id: "garden", title: "Garden", icon: null, color: null, parentId: "home" },
];

const notes = [
	note("groceries", 30, 0.1, { categoryId: "garden" }),
	note("plumber", 1, 1),
	note("sprint", 3, 3),
	note("a", 3, 3),
];
const ids = (list: { id?: string }[]) => list.map((n) => n.id);

describe("recentNotes", () => {
	it("orders by creation time, then by ID descending", () => {
		expect(
			ids(recentNotes(notes, categories, "created", 10, null, now)),
		).toEqual(["plumber", "sprint", "a", "groceries"]);
	});

	it("orders by time of last change", () => {
		expect(
			ids(recentNotes(notes, categories, "updated", 10, null, now)),
		).toEqual(["groceries", "plumber", "sprint", "a"]);
	});

	it("limits by count and days of the chosen time", () => {
		expect(
			ids(recentNotes(notes, categories, "created", 2, null, now)),
		).toEqual(["plumber", "sprint"]);
		expect(ids(recentNotes(notes, categories, "created", 10, 7, now))).toEqual([
			"plumber",
			"sprint",
			"a",
		]);
		expect(ids(recentNotes(notes, categories, "updated", 10, 1, now))).toEqual([
			"groceries",
			"plumber",
		]);
	});

	it("shows the display title, the chosen time and the category with its path", () => {
		const untitled = note("x", 2, 1, {
			title: null,
			content: {
				v: 1,
				blocks: [
					{
						type: "paragraph",
						content: [{ type: "text", text: "call the plumber" }],
					},
				],
			},
		});
		const [row] = recentNotes([untitled], categories, "updated", 10, null, now);
		expect(row).toEqual({
			id: "x",
			title: "call the plumber",
			icon: null,
			color: null,
			pinned: false,
			archived: false,
			time: daysAgo(1),
			category: null,
		});
		const [groceries] = recentNotes(notes, categories, "updated", 1, null, now);
		expect(groceries.category).toEqual({
			title: "Garden",
			path: ["Home", "Garden"],
		});
	});
});
