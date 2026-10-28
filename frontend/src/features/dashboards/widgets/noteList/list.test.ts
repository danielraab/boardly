import { describe, expect, it } from "vitest";
import type { Note, NoteCategory } from "#/features/notes/api";
import { type ListedNote, noteList } from "./list";

const base = Date.parse("2026-10-01T12:00:00Z");
const hour = 60 * 60 * 1000;
const at = (n: number) => new Date(base + n * hour).toISOString();

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
	version: 1,
	createdAt: at(created),
	updatedAt: at(updated),
	...fields,
});

const categories: NoteCategory[] = [
	{ id: "home", title: "Home", icon: null, color: null, parentId: null },
	{ id: "garden", title: "Garden", icon: null, color: null, parentId: "home" },
	{ id: "work", title: "Work", icon: null, color: null, parentId: null },
];

// As in the Go test: Plumber in Home, Roses in Garden, Draft uncategorized,
// created in this order and Plumber changed last.
const notes = [
	note("Plumber", 0, 3, { categoryId: "home" }),
	note("Roses", 1, 1, { categoryId: "garden" }),
	note("Draft", 2, 2),
];

const config = {
	category: "all" as const,
	categoryId: null,
	subcategories: true,
	sort: "updated" as const,
	limit: 10,
};

const titles = (data: ReturnType<typeof noteList>) =>
	"notes" in data ? data.notes.map((n) => n.title) : data;

describe("noteList", () => {
	it.each([
		["all by updated", {}, ["Plumber", "Draft", "Roses"]],
		["all by created", { sort: "created" }, ["Draft", "Roses", "Plumber"]],
		["all by title", { sort: "title" }, ["Draft", "Plumber", "Roses"]],
		["limit", { sort: "created", limit: 2 }, ["Draft", "Roses"]],
		["title limit", { sort: "title", limit: 2 }, ["Draft", "Plumber"]],
		["uncategorized", { category: "none" }, ["Draft"]],
		[
			"with subcategories",
			{ category: "category", categoryId: "home", sort: "created" },
			["Roses", "Plumber"],
		],
		[
			"without subcategories",
			{ category: "category", categoryId: "home", subcategories: false },
			["Plumber"],
		],
		["empty category", { category: "category", categoryId: "work" }, []],
	] as const)("%s", (_, overrides, want) => {
		expect(
			titles(noteList(notes, categories, { ...config, ...overrides })),
		).toEqual(want);
	});

	it("reports a missing category", () => {
		expect(
			noteList(notes, categories, {
				...config,
				category: "category",
				categoryId: "gone",
			}),
		).toEqual({ missing: true });
	});

	it("sorts by display title ignoring case and accents, untitled last", () => {
		const text = (s: string) => ({
			v: 1 as const,
			blocks: [
				{
					type: "paragraph" as const,
					content: [{ type: "text" as const, text: s }],
				},
			],
		});
		const list = [
			note("1", 0, 0, { title: "zoo" }),
			note("2", 0, 0, { title: null }),
			note("3", 0, 0, { title: "Ärzte" }),
			note("4", 0, 0, { title: null, content: text("beta") }),
			note("0", 0, 0, { title: null }),
		];
		const data = noteList(list, [], { ...config, sort: "title" });
		expect("notes" in data && data.notes.map((n) => n.id)).toEqual([
			"3",
			"4",
			"1",
			"0",
			"2",
		]);
	});

	it("names only categories other than the chosen one", () => {
		const second = (data: ReturnType<typeof noteList>) =>
			"notes" in data
				? Object.fromEntries(
						data.notes.map((n: ListedNote) => [n.title, n.category]),
					)
				: data;
		expect(
			second(
				noteList(notes, categories, {
					...config,
					category: "category",
					categoryId: "home",
				}),
			),
		).toEqual({
			Plumber: null,
			Roses: { title: "Garden", path: ["Home", "Garden"] },
		});
		expect(second(noteList(notes, categories, config))).toEqual({
			Plumber: { title: "Home", path: ["Home"] },
			Roses: { title: "Garden", path: ["Home", "Garden"] },
			Draft: null,
		});
	});
});
