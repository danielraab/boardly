import { describe, expect, it } from "vitest";
import type { FavouriteNode } from "./api";
import {
	applyMove,
	currentSlot,
	filterTree,
	flatten,
	folderPath,
	highlight,
	indicatorFor,
	keyboardSlots,
	linkCounts,
	pointerSlot,
	ROOT_END,
	searchTerms,
	tagCounts,
} from "./tree";

const link = (id: string): FavouriteNode => ({
	id,
	kind: "link",
	title: id,
	url: `https://example.org/${id}`,
});
const folder = (id: string, ...children: FavouriteNode[]): FavouriteNode => ({
	id,
	kind: "folder",
	title: id,
	children,
});

// root
// ├─ A
// ├─ Work
// │  ├─ Tools
// │  │  └─ Git
// │  └─ Docs
// ├─ Empty
// └─ B
const tree: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		link("A"),
		folder("Work", folder("Tools", link("Git")), link("Docs")),
		folder("Empty"),
		link("B"),
	],
};
const none = new Set<string>();

const titles = (n: FavouriteNode | undefined) =>
	(n?.children ?? []).map((c) => c.title);

describe("flatten", () => {
	it("lists visible rows with depth", () => {
		expect(flatten(tree, none).map((i) => `${i.depth}:${i.node.id}`)).toEqual([
			"0:A",
			"0:Work",
			"1:Tools",
			"2:Git",
			"1:Docs",
			"0:Empty",
			"0:B",
		]);
	});

	it("hides the contents of collapsed folders", () => {
		expect(flatten(tree, new Set(["Work"])).map((i) => i.node.id)).toEqual([
			"A",
			"Work",
			"Empty",
			"B",
		]);
	});
});

describe("pointerSlot", () => {
	const at = (overId: string, fraction: number, dragged = "B") =>
		pointerSlot(tree, none, dragged, overId, fraction);

	it("drops before or after a link by halves", () => {
		expect(at("A", 0.2, "B")).toEqual({ parentId: "root", position: 0 });
		expect(at("A", 0.8, "B")).toEqual({ parentId: "root", position: 1 });
	});

	it("counts positions without the dragged node", () => {
		// A is dragged: Work becomes index 0 among the remaining root children.
		expect(at("Work", 0.1, "A")).toEqual({ parentId: "root", position: 0 });
		expect(at("Empty", 0.9, "A")).toEqual({ parentId: "root", position: 2 });
	});

	it("drops into the middle of a folder, at its end", () => {
		expect(at("Work", 0.5)).toEqual({
			parentId: "Work",
			position: 2,
			into: true,
		});
		expect(at("Empty", 0.5)).toEqual({
			parentId: "Empty",
			position: 0,
			into: true,
		});
	});

	it("drops below an expanded folder with children at its first position", () => {
		expect(at("Work", 0.9)).toEqual({ parentId: "Work", position: 0 });
		expect(pointerSlot(tree, new Set(["Work"]), "B", "Work", 0.9)).toEqual({
			parentId: "root",
			position: 2,
		});
	});

	it("offers no slot inside the dragged folder's own subtree", () => {
		expect(at("Work", 0.5, "Work")).toBeNull();
		expect(at("Tools", 0.5, "Work")).toBeNull();
		expect(at("Git", 0.2, "Work")).toBeNull();
	});

	it("maps the area below the tree to the end of the root", () => {
		expect(at(ROOT_END, 0.5, "A")).toEqual({ parentId: "root", position: 3 });
	});
});

describe("keyboardSlots", () => {
	const key = (s: { parentId: string; position: number; into?: boolean }) =>
		`${s.parentId}:${s.position}${s.into ? "*" : ""}`;

	it("lists every slot top to bottom without the dragged node", () => {
		expect(keyboardSlots(tree, none, "Docs").map(key)).toEqual([
			"root:0",
			"root:1",
			"Work:0",
			"Tools:0",
			"Tools:1",
			"Work:1",
			"root:2",
			"Empty:0",
			"root:3",
			"root:4",
		]);
	});

	it("skips the dragged folder's subtree and enters collapsed folders at their end", () => {
		expect(keyboardSlots(tree, new Set(["Work"]), "A").map(key)).toEqual([
			"root:0",
			"Work:2*",
			"root:1",
			"Empty:0",
			"root:2",
			"root:3",
		]);
		expect(keyboardSlots(tree, none, "Work").map(key)).toEqual([
			"root:0",
			"root:1",
			"Empty:0",
			"root:2",
			"root:3",
		]);
	});

	it("contains the current slot of the dragged node", () => {
		const slots = keyboardSlots(tree, none, "Docs").map(key);
		const current = currentSlot(tree, "Docs");
		expect(current).toEqual({ parentId: "Work", position: 1 });
		expect(slots).toContain(key(current ?? { parentId: "", position: 0 }));
	});
});

describe("indicatorFor", () => {
	const show = (slot: { parentId: string; position: number; into?: boolean }) =>
		indicatorFor(tree, none, "B", slot);

	it("draws a line above the node that will follow", () => {
		expect(show({ parentId: "Work", position: 1 })).toEqual({
			rowId: "Docs",
			edge: "top",
			depth: 1,
		});
	});

	it("draws the end of a folder below its last visible row", () => {
		expect(show({ parentId: "Tools", position: 1 })).toEqual({
			rowId: "Git",
			edge: "bottom",
			depth: 2,
		});
		expect(show({ parentId: "root", position: 3 })).toEqual({
			rowId: "Empty",
			edge: "bottom",
			depth: 0,
		});
	});

	it("highlights a folder for drops into it", () => {
		expect(show({ parentId: "Work", position: 2, into: true })).toEqual({
			rowId: "Work",
			edge: "inside",
			depth: 1,
		});
		expect(show({ parentId: "Empty", position: 0 })).toEqual({
			rowId: "Empty",
			edge: "inside",
			depth: 1,
		});
	});
});

describe("applyMove", () => {
	it("reorders within a folder", () => {
		const moved = applyMove(tree, "B", "root", 0);
		expect(titles(moved)).toEqual(["B", "A", "Work", "Empty"]);
	});

	it("moves a folder with its contents into another folder", () => {
		const moved = applyMove(tree, "Work", "Empty", 0);
		expect(titles(moved)).toEqual(["A", "Empty", "B"]);
		const empty = moved.children?.[1];
		expect(titles(empty)).toEqual(["Work"]);
		expect(titles(empty?.children?.[0])).toEqual(["Tools", "Docs"]);
	});

	it("leaves the original tree unchanged", () => {
		applyMove(tree, "A", "Work", 0);
		expect(titles(tree)).toEqual(["A", "Work", "Empty", "B"]);
	});
});

// root
// ├─ Work
// │  ├─ Tools
// │  │  ├─ Grafana (monitoring, Tools)   https://grafana.example.com
// │  │  └─ Empty
// │  └─ Mail (reading)
// ├─ Grafana                          https://other.example.org
// └─ Long read (reading, later)
const tagged: FavouriteNode = {
	id: "root",
	kind: "folder",
	title: null,
	children: [
		folder(
			"Work",
			folder(
				"Tools",
				{
					id: "graf",
					kind: "link",
					title: "Grafana",
					url: "https://grafana.example.com",
					tags: ["monitoring", "Tools"],
				},
				folder("Empty"),
			),
			{ ...link("Mail"), tags: ["reading"] },
		),
		{
			id: "graf2",
			kind: "link",
			title: "Grafana",
			url: "https://other.example.org",
		},
		{ ...link("Long read"), tags: ["reading", "later"] },
	],
};

const ids = (n: FavouriteNode | null) =>
	n ? flatten(n, none).map((i) => i.node.id) : null;
const search = (text: string, tags: string[] = []) =>
	ids(filterTree(tagged, { terms: searchTerms(text), tags }));

describe("filterTree", () => {
	it("returns the tree unchanged without a filter", () => {
		expect(filterTree(tagged, { terms: [], tags: [] })).toBe(tagged);
	});

	it("finds links by title with their folders", () => {
		expect(search("GRAF")).toEqual(["Work", "Tools", "graf", "graf2"]);
	});

	it("finds links by URL and tag", () => {
		expect(search("grafana.example")).toEqual(["Work", "Tools", "graf"]);
		expect(search("monitor")).toEqual(["Work", "Tools", "graf"]);
	});

	it("needs every term, also from folder names on the path", () => {
		expect(search("work graf")).toEqual(["Work", "Tools", "graf"]);
	});

	it("keeps a matching folder with everything inside it", () => {
		expect(search("tools")).toEqual(["Work", "Tools", "graf", "Empty"]);
	});

	it("needs every chosen tag, ignoring case", () => {
		expect(search("", ["reading"])).toEqual(["Work", "Mail", "Long read"]);
		expect(search("", ["Reading", "later"])).toEqual(["Long read"]);
	});

	it("doesn't keep whole folders by name while filtering by tag", () => {
		expect(search("tools", ["reading"])).toBeNull();
		expect(search("work", ["reading"])).toEqual(["Work", "Mail"]);
	});

	it("returns null when nothing matches", () => {
		expect(search("nothing")).toBeNull();
	});
});

describe("tagCounts", () => {
	it("counts links per tag, sorted by name", () => {
		expect(tagCounts(tagged)).toEqual([
			{ name: "later", count: 1 },
			{ name: "monitoring", count: 1 },
			{ name: "reading", count: 2 },
			{ name: "Tools", count: 1 },
		]);
	});
});

describe("linkCounts", () => {
	it("counts links at all levels below a folder", () => {
		const counts = linkCounts(tagged);
		expect(counts.get("root")).toBe(4);
		expect(counts.get("Work")).toBe(2);
		expect(counts.get("Tools")).toBe(1);
		expect(counts.get("Empty")).toBe(0);
	});
});

describe("folderPath", () => {
	it("lists the folders down to a node", () => {
		expect(folderPath(tagged, "Empty")).toEqual(["Work", "Tools", "Empty"]);
		expect(folderPath(tagged, "unknown")).toEqual([]);
	});
});

describe("highlight", () => {
	it("marks matches ignoring case and merges overlaps", () => {
		expect(highlight("Grafana", ["afa", "fan"])).toEqual([
			{ text: "Gr", match: false },
			{ text: "afan", match: true },
			{ text: "a", match: false },
		]);
	});

	it("returns the text unmarked without terms", () => {
		expect(highlight("Mail", [])).toEqual([{ text: "Mail", match: false }]);
	});
});
