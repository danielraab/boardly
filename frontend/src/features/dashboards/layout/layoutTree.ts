import { isSection, type Section, type Widget, withoutData } from "../api";

/** Sections nest at most this many levels; a top-level section is level 1. */
export const maxDepth = 5;

export type SectionSettings = Omit<Section, "id" | "kind" | "children">;

export type LayoutAction =
	| { type: "reset"; sections: Widget[] }
	| { type: "add"; parentId: string | null; widget: Widget }
	| { type: "update"; id: string; widget: Widget }
	| { type: "remove"; id: string }
	| { type: "move"; id: string; parentId: string | null; index: number };

/**
 * Why a widget can't go into a parent, or null if it can. "topLevel": only
 * sections can be at the top level.
 */
export type MoveProblem = "tooDeep" | "intoItself" | "topLevel";

/** The pure reducer of the editor's draft; unknown IDs leave the tree as it is. */
export function layoutReducer(tree: Widget[], action: LayoutAction): Widget[] {
	switch (action.type) {
		case "reset":
			return action.sections;
		case "add":
			return insertAt(tree, action.parentId, Infinity, action.widget);
		case "update":
			return mapWidget(tree, action.id, (w) =>
				isSection(w) && isSection(action.widget)
					? { ...action.widget, id: w.id, children: w.children }
					: { ...action.widget, id: w.id },
			);
		case "remove":
			return removeWidget(tree, action.id)[0];
		case "move": {
			if (canMove(tree, action.id, action.parentId) !== null) return tree;
			const [rest, moved] = removeWidget(tree, action.id);
			if (!moved) return tree;
			return insertAt(rest, action.parentId, action.index, moved);
		}
	}
}

/** The children of a widget; content widgets have none. */
export function childrenOf(widget: Widget): Widget[] {
	return isSection(widget) ? widget.children : [];
}

export function findWidget(tree: Widget[], id: string): Widget | undefined {
	for (const w of tree) {
		if (w.id === id) return w;
		const found = findWidget(childrenOf(w), id);
		if (found) return found;
	}
	return undefined;
}

export function findSection(tree: Widget[], id: string): Section | undefined {
	const widget = findWidget(tree, id);
	return widget && isSection(widget) ? widget : undefined;
}

/** The parent's ID of a widget (null at the top level), undefined if unknown. */
export function parentOf(
	tree: Widget[],
	id: string,
	parentId: string | null = null,
): string | null | undefined {
	for (const w of tree) {
		if (w.id === id) return parentId;
		const found = parentOf(childrenOf(w), id, w.id);
		if (found !== undefined) return found;
	}
	return undefined;
}

/** The level of a section (1 at the top), or 0 for null (the dashboard itself). */
export function depthOf(tree: Widget[], id: string | null): number {
	if (id === null) return 0;
	const walk = (list: Widget[], level: number): number => {
		for (const w of list) {
			if (w.id === id) return level;
			const found = walk(childrenOf(w), level + 1);
			if (found) return found;
		}
		return 0;
	};
	return walk(tree, 1);
}

/**
 * The number of levels a widget takes up, counting itself. Content widgets
 * take none, because only sections count as nesting levels.
 */
export function subtreeHeight(widget: Widget): number {
	if (!isSection(widget)) return 0;
	return 1 + Math.max(0, ...widget.children.map(subtreeHeight));
}

/** Whether id is the widget ancestorId itself or inside it. */
export function isWithin(tree: Widget[], id: string, ancestorId: string) {
	const ancestor = findWidget(tree, ancestorId);
	return ancestor !== undefined && findWidget([ancestor], id) !== undefined;
}

/** Checks whether widget id can be put into parentId (null: the top level). */
export function canMove(
	tree: Widget[],
	id: string,
	parentId: string | null,
): MoveProblem | null {
	const widget = findWidget(tree, id);
	if (!widget) return null;
	if (parentId === null && !isSection(widget)) return "topLevel";
	if (parentId !== null && isWithin(tree, parentId, id)) return "intoItself";
	if (depthOf(tree, parentId) + subtreeHeight(widget) > maxDepth)
		return "tooDeep";
	return null;
}

/** The number of widgets inside a widget, at any level. */
export function countDescendants(widget: Widget): number {
	return childrenOf(widget).reduce((n, c) => n + 1 + countDescendants(c), 0);
}

/** Compares two trees, ignoring the data the server resolved. */
export function isSameTree(a: Widget[], b: Widget[]) {
	return JSON.stringify(withoutData(a)) === JSON.stringify(withoutData(b));
}

function mapWidget(
	tree: Widget[],
	id: string,
	fn: (w: Widget) => Widget,
): Widget[] {
	return tree.map((w) => {
		if (w.id === id) return fn(w);
		if (!isSection(w)) return w;
		return { ...w, children: mapWidget(w.children, id, fn) };
	});
}

/** Returns the tree without the widget and the removed widget. */
function removeWidget(
	tree: Widget[],
	id: string,
): [Widget[], Widget | undefined] {
	let removed: Widget | undefined;
	const walk = (list: Widget[]): Widget[] =>
		list.flatMap((w) => {
			if (w.id === id) {
				removed = w;
				return [];
			}
			return [isSection(w) ? { ...w, children: walk(w.children) } : w];
		});
	const rest = walk(tree);
	return [rest, removed];
}

function insertAt(
	tree: Widget[],
	parentId: string | null,
	index: number,
	widget: Widget,
): Widget[] {
	const insert = (list: Widget[]) => {
		const copy = [...list];
		copy.splice(Math.min(Math.max(index, 0), copy.length), 0, widget);
		return copy;
	};
	if (parentId === null) return insert(tree);
	return mapWidget(tree, parentId, (p) =>
		isSection(p) ? { ...p, children: insert(p.children) } : p,
	);
}

/** A place in the tree: an index in the children of parentId (null: the top level). */
export type Slot = { parentId: string | null; index: number };

/** The children of parentId (null: the top level). */
export function listOf(tree: Widget[], parentId: string | null): Widget[] {
	if (parentId === null) return tree;
	return findSection(tree, parentId)?.children ?? [];
}

/** Where a widget is now, as a slot. */
export function currentSlot(tree: Widget[], id: string): Slot | undefined {
	const parentId = parentOf(tree, id);
	if (parentId === undefined) return undefined;
	const list = listOf(tree, parentId);
	return { parentId, index: list.findIndex((w) => w.id === id) };
}

/**
 * Every place widget id could be moved to, in reading order: before each
 * widget, inside each section, and at the end of each list. Indexes count
 * without the moved widget, as `move` expects them. Places that are too deep
 * are included, so moving there can be refused with a message. A content
 * widget gets no places at the top level.
 */
export function keyboardSlots(tree: Widget[], id: string): Slot[] {
	const [rest, moved] = removeWidget(tree, id);
	const topLevel = moved === undefined || isSection(moved);
	const slots: Slot[] = [];
	const walk = (list: Widget[], parentId: string | null) => {
		for (let i = 0; i <= list.length; i++) {
			if (parentId !== null || topLevel) slots.push({ parentId, index: i });
			const next = list[i];
			if (next && isSection(next)) walk(next.children, next.id);
		}
	};
	walk(rest, null);
	return slots;
}
