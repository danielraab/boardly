import { describe, expect, it } from "vitest";
import type { Note, NoteCategory } from "#/features/notes/api";
import { noteChoices, selectedNotes } from "./selected";

const note = (id: string, fields: Partial<Note> = {}): Note => ({
	id,
	title: id,
	icon: null,
	color: null,
	categoryId: null,
	content: { v: 1, blocks: [] },
	version: 1,
	createdAt: "2026-10-01T12:00:00Z",
	updatedAt: "2026-10-01T12:00:00Z",
	...fields,
});

const categories: NoteCategory[] = [
	{ id: "home", title: "Home", icon: null, color: null, parentId: null },
	{ id: "garden", title: "Garden", icon: null, color: null, parentId: "home" },
];

// As in the Go test: an untitled plumber note, Roses in Garden, Draft in Home.
const notes = [
	note("plumber", {
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
	}),
	note("Roses", {
		categoryId: "garden",
		icon: { type: "symbol", name: "shopping-cart" },
	}),
	note("Draft", { categoryId: "home" }),
];

describe("selectedNotes", () => {
	it("keeps the order of the list and skips missing notes", () => {
		const data = selectedNotes(notes, categories, {
			noteIds: ["Roses", "gone", "plumber"],
		});
		expect(data).toEqual({
			notes: [
				{
					id: "Roses",
					title: "Roses",
					icon: { type: "symbol", name: "shopping-cart" },
					category: { title: "Garden", path: ["Home", "Garden"] },
				},
				{
					id: "plumber",
					title: "call the plumber",
					icon: null,
					category: null,
				},
			],
		});
	});

	it("names the category of every note that has one", () => {
		const data = selectedNotes(notes, categories, { noteIds: ["Draft"] });
		expect(data).toEqual({
			notes: [
				{
					id: "Draft",
					title: "Draft",
					icon: null,
					category: { title: "Home", path: ["Home"] },
				},
			],
		});
	});

	it("is missing when no note exists", () => {
		expect(
			selectedNotes(notes, categories, { noteIds: ["gone", "also gone"] }),
		).toEqual({ missing: true });
	});
});

describe("noteChoices", () => {
	const all = [...notes, note("Ärzte"), note("zoo")];
	const titles = (query: string, chosen: string[] = []) =>
		noteChoices(all, categories, chosen, query).map((c) => c.title);

	it("sorts by display title with untitled notes last", () => {
		expect(titles("")).toEqual([
			"Ärzte",
			"call the plumber",
			"Draft",
			"Roses",
			"zoo",
		]);
	});

	it("matches the title ignoring case and accents", () => {
		expect(titles("ARZ")).toEqual(["Ärzte"]);
		expect(titles("plum")).toEqual(["call the plumber"]);
	});

	it("matches the category path", () => {
		expect(titles("garden")).toEqual(["Roses"]);
		expect(titles("home")).toEqual(["Draft", "Roses"]);
		expect(noteChoices(all, categories, [], "rose")[0].path).toEqual([
			"Home",
			"Garden",
		]);
	});

	it("leaves out chosen notes", () => {
		expect(titles("", ["Roses", "zoo"])).toEqual([
			"Ärzte",
			"call the plumber",
			"Draft",
		]);
	});
});
