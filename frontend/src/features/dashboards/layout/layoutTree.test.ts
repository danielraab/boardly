import { describe, expect, it } from "vitest";
import { ApiError } from "#/lib/api";
import {
	type ContentWidget,
	isLayoutConflict,
	isSection,
	type Section,
	type Widget,
} from "../api";
import {
	canMove,
	countDescendants,
	currentSlot,
	depthOf,
	findWidget,
	isSameTree,
	keyboardSlots,
	layoutReducer,
	parentOf,
	subtreeHeight,
} from "./layoutTree";

const s = (id: string, ...children: Widget[]): Section => ({
	id,
	kind: "section",
	width: 6,
	title: id,
	border: true,
	height: "auto",
	children,
});

/** Renders a tree as "a(b,c),d". */
const shape = (tree: Widget[]): string =>
	tree
		.map((x) =>
			isSection(x) && x.children.length
				? `${x.id}(${shape(x.children)})`
				: x.id,
		)
		.join(",");

/** A content widget of the kind "note". */
const w = (id: string, data?: unknown): ContentWidget => ({
	id,
	kind: "note",
	width: 2,
	height: "auto",
	config: { text: id },
	...(data === undefined ? {} : { data }),
});

/** A chain of sections c1 > c2 > … > c<n>. */
const chain = (n: number, from = 1): Section[] =>
	n < from ? [] : [s(`c${from}`, ...chain(n, from + 1))];

describe("layoutReducer", () => {
	const tree = [s("a", s("b"), s("c")), s("d")];

	it("adds at the end of the top level or a parent", () => {
		expect(
			shape(
				layoutReducer(tree, { type: "add", parentId: null, widget: s("x") }),
			),
		).toBe("a(b,c),d,x");
		expect(
			shape(
				layoutReducer(tree, { type: "add", parentId: "d", widget: s("x") }),
			),
		).toBe("a(b,c),d(x)");
	});

	it("updates settings without touching the children", () => {
		const next = layoutReducer(tree, {
			type: "update",
			id: "a",
			widget: { ...s("a"), border: false, height: 4, children: [] },
		});
		expect(next[0]).toMatchObject({ border: false, height: 4 });
		expect(shape(next)).toBe("a(b,c),d");
	});

	it("removes a section with its subtree", () => {
		expect(shape(layoutReducer(tree, { type: "remove", id: "a" }))).toBe("d");
	});

	it("moves a section with its children", () => {
		expect(
			shape(
				layoutReducer(tree, { type: "move", id: "a", parentId: "d", index: 0 }),
			),
		).toBe("d(a(b,c))");
		expect(
			shape(
				layoutReducer(tree, {
					type: "move",
					id: "c",
					parentId: null,
					index: 0,
				}),
			),
		).toBe("c,a(b),d");
		expect(
			shape(
				layoutReducer(tree, { type: "move", id: "b", parentId: "a", index: 1 }),
			),
		).toBe("a(c,b),d");
	});

	it("refuses to move a section into itself or its descendants", () => {
		expect(
			layoutReducer(tree, { type: "move", id: "a", parentId: "b", index: 0 }),
		).toBe(tree);
		expect(
			layoutReducer(tree, { type: "move", id: "a", parentId: "a", index: 0 }),
		).toBe(tree);
	});
});

describe("tree helpers", () => {
	it("knows depths, heights and parents", () => {
		const tree = chain(3);
		expect(depthOf(tree, null)).toBe(0);
		expect(depthOf(tree, "c3")).toBe(3);
		expect(subtreeHeight(tree[0])).toBe(3);
		expect(parentOf(tree, "c3")).toBe("c2");
		expect(parentOf(tree, "c1")).toBeNull();
		expect(parentOf(tree, "nope")).toBeUndefined();
		expect(countDescendants(tree[0])).toBe(2);
	});

	it("allows 5 levels and refuses 6", () => {
		// x has one child, so it takes up two levels.
		const tree = [...chain(4), s("x", s("y"))];
		expect(canMove(tree, "x", "c3")).toBeNull();
		expect(canMove(tree, "x", "c4")).toBe("tooDeep");
		expect(canMove(tree, "c1", "x")).toBeNull();
		expect(canMove(tree, "c1", "y")).toBe("tooDeep");
		expect(canMove(tree, "c1", "c2")).toBe("intoItself");
	});

	it("compares trees, ignoring resolved data", () => {
		expect(isSameTree(chain(2), chain(2))).toBe(true);
		expect(isSameTree(chain(2), chain(3))).toBe(false);
		expect(isSameTree([s("a", w("n", { tree: 1 }))], [s("a", w("n"))])).toBe(
			true,
		);
	});

	it("treats content widgets as leaves that add no level", () => {
		const tree = [...chain(5), s("x", w("n"))];
		expect(subtreeHeight(tree[1])).toBe(1);
		expect(countDescendants(tree[1])).toBe(1);
		// A widget fits into a section at level 5, but not at the top level.
		expect(canMove(tree, "n", "c5")).toBeNull();
		expect(canMove(tree, "n", null)).toBe("topLevel");
		expect(
			shape(
				layoutReducer(tree, {
					type: "move",
					id: "n",
					parentId: "c5",
					index: 0,
				}),
			),
		).toBe("c1(c2(c3(c4(c5(n))))),x");
		expect(
			layoutReducer(tree, { type: "move", id: "n", parentId: null, index: 0 }),
		).toBe(tree);
	});

	it("updates a content widget in place", () => {
		const tree = [s("a", w("n"))];
		const next = layoutReducer(tree, {
			type: "update",
			id: "n",
			widget: { ...w("n"), height: 3, config: { text: "new" } },
		});
		expect(findWidget(next, "n")).toMatchObject({
			height: 3,
			config: { text: "new" },
		});
	});
});

describe("isLayoutConflict", () => {
	it("detects 409 responses only", () => {
		expect(isLayoutConflict(new ApiError(409, "layout_conflict", ""))).toBe(
			true,
		);
		expect(isLayoutConflict(new ApiError(400, "invalid_layout", ""))).toBe(
			false,
		);
		expect(isLayoutConflict(new Error("offline"))).toBe(false);
	});
});

describe("keyboard slots", () => {
	it("lists every place in reading order, without the moved section", () => {
		const tree = [s("a", s("b")), s("c")];
		const slots = keyboardSlots(tree, "c").map(
			(x) => `${x.parentId ?? "root"}:${x.index}`,
		);
		expect(slots).toEqual(["root:0", "a:0", "b:0", "a:1", "root:1"]);
		expect(currentSlot(tree, "c")).toEqual({ parentId: null, index: 1 });
		expect(currentSlot(tree, "b")).toEqual({ parentId: "a", index: 0 });
	});

	it("leaves out the moved section's own subtree", () => {
		const tree = [s("a", s("b")), s("c")];
		const slots = keyboardSlots(tree, "a").map(
			(x) => `${x.parentId ?? "root"}:${x.index}`,
		);
		expect(slots).toEqual(["root:0", "c:0", "root:1"]);
	});

	it("gives a content widget no places at the top level", () => {
		const tree = [s("a", w("n"), s("b")), s("c")];
		const slots = keyboardSlots(tree, "n").map(
			(x) => `${x.parentId ?? "root"}:${x.index}`,
		);
		expect(slots).toEqual(["a:0", "b:0", "a:1", "c:0"]);
	});
});
