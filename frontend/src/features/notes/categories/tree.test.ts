import { describe, expect, it } from "vitest";
import type { Note, NoteCategory } from "../api";
import {
	buildTree,
	categoryPath,
	filterNotes,
	flatten,
	subtreeIds,
} from "./tree";

const cat = (
	id: string,
	title: string,
	parentId: string | null = null,
): NoteCategory => ({
	id,
	title,
	parentId,
	icon: null,
	color: null,
});

const note = (id: string, categoryId: string | null) =>
	({ id, categoryId }) as Note;

const categories = [
	cat("w", "work"),
	cat("h", "Home"),
	cat("g", "Garden", "h"),
	cat("b", "Beds", "g"),
	cat("a", "Ärzte"),
	cat("o", "Orphan", "missing"),
];

describe("category tree", () => {
	it("nests and sorts by title, ignoring case, in the viewer's language", () => {
		const titles = flatten(buildTree(categories, "de")).map(
			(n) => `${"-".repeat(n.depth)}${n.category.title}`,
		);
		expect(titles).toEqual([
			"Ärzte",
			"Home",
			"-Garden",
			"--Beds",
			"Orphan",
			"work",
		]);
	});

	it("collects subcategories at every depth", () => {
		expect([...subtreeIds(categories, "h")].sort()).toEqual(["b", "g", "h"]);
	});

	it("builds the path of a category", () => {
		expect(categoryPath(categories, "b")).toEqual(["Home", "Garden", "Beds"]);
		expect(categoryPath(categories, "nope")).toEqual([]);
	});

	it("filters notes by category including subcategories", () => {
		const notes = [
			note("1", "h"),
			note("2", "g"),
			note("3", "b"),
			note("4", null),
			note("5", "w"),
		];
		const ids = (f: Parameters<typeof filterNotes>[2]) =>
			filterNotes(notes, categories, f).map((n) => n.id);
		expect(ids("all")).toEqual(["1", "2", "3", "4", "5"]);
		expect(ids("none")).toEqual(["4"]);
		expect(ids({ id: "h" })).toEqual(["1", "2", "3"]);
		expect(ids({ id: "g" })).toEqual(["2", "3"]);
	});
});
