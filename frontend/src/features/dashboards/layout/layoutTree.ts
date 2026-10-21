import type { Section } from "../api";

/** Sections nest at most this many levels; a top-level section is level 1. */
export const maxDepth = 5;

export type SectionSettings = Omit<Section, "id" | "children">;

export type LayoutAction =
	| { type: "reset"; sections: Section[] }
	| { type: "add"; parentId: string | null; section: Section }
	| { type: "update"; id: string; settings: Partial<SectionSettings> }
	| { type: "remove"; id: string }
	| { type: "move"; id: string; parentId: string | null; index: number };

/** Why a section can't go into a parent, or null if it can. */
export type MoveProblem = "tooDeep" | "intoItself";

/** The pure reducer of the editor's draft; unknown IDs leave the tree as it is. */
export function layoutReducer(
	tree: Section[],
	action: LayoutAction,
): Section[] {
	switch (action.type) {
		case "reset":
			return action.sections;
		case "add":
			return insertAt(tree, action.parentId, Infinity, action.section);
		case "update":
			return mapSection(tree, action.id, (s) => ({ ...s, ...action.settings }));
		case "remove":
			return removeSection(tree, action.id)[0];
		case "move": {
			if (canMove(tree, action.id, action.parentId) !== null) return tree;
			const [rest, moved] = removeSection(tree, action.id);
			if (!moved) return tree;
			return insertAt(rest, action.parentId, action.index, moved);
		}
	}
}

export function findSection(tree: Section[], id: string): Section | undefined {
	for (const s of tree) {
		if (s.id === id) return s;
		const found = findSection(s.children, id);
		if (found) return found;
	}
	return undefined;
}

/** The parent's ID of a section (null at the top level), undefined if unknown. */
export function parentOf(
	tree: Section[],
	id: string,
	parentId: string | null = null,
): string | null | undefined {
	for (const s of tree) {
		if (s.id === id) return parentId;
		const found = parentOf(s.children, id, s.id);
		if (found !== undefined) return found;
	}
	return undefined;
}

/** The level of a section (1 at the top), or 0 for null (the dashboard itself). */
export function depthOf(tree: Section[], id: string | null): number {
	if (id === null) return 0;
	const walk = (list: Section[], level: number): number => {
		for (const s of list) {
			if (s.id === id) return level;
			const found = walk(s.children, level + 1);
			if (found) return found;
		}
		return 0;
	};
	return walk(tree, 1);
}

/** The number of levels a section takes up, counting itself. */
export function subtreeHeight(section: Section): number {
	return 1 + Math.max(0, ...section.children.map(subtreeHeight));
}

/** Whether id is the section ancestorId itself or inside it. */
export function isWithin(tree: Section[], id: string, ancestorId: string) {
	const ancestor = findSection(tree, ancestorId);
	return ancestor !== undefined && findSection([ancestor], id) !== undefined;
}

/** Checks whether section id can be put into parentId (null: the top level). */
export function canMove(
	tree: Section[],
	id: string,
	parentId: string | null,
): MoveProblem | null {
	const section = findSection(tree, id);
	if (!section) return null;
	if (parentId !== null && isWithin(tree, parentId, id)) return "intoItself";
	if (depthOf(tree, parentId) + subtreeHeight(section) > maxDepth)
		return "tooDeep";
	return null;
}

/** The number of sections inside a section, at any level. */
export function countDescendants(section: Section): number {
	return section.children.reduce((n, c) => n + 1 + countDescendants(c), 0);
}

export function isSameTree(a: Section[], b: Section[]) {
	return JSON.stringify(a) === JSON.stringify(b);
}

function mapSection(
	tree: Section[],
	id: string,
	fn: (s: Section) => Section,
): Section[] {
	return tree.map((s) =>
		s.id === id ? fn(s) : { ...s, children: mapSection(s.children, id, fn) },
	);
}

/** Returns the tree without the section and the removed section. */
function removeSection(
	tree: Section[],
	id: string,
): [Section[], Section | undefined] {
	let removed: Section | undefined;
	const walk = (list: Section[]): Section[] =>
		list.flatMap((s) => {
			if (s.id === id) {
				removed = s;
				return [];
			}
			return [{ ...s, children: walk(s.children) }];
		});
	const rest = walk(tree);
	return [rest, removed];
}

function insertAt(
	tree: Section[],
	parentId: string | null,
	index: number,
	section: Section,
): Section[] {
	const insert = (list: Section[]) => {
		const copy = [...list];
		copy.splice(Math.min(Math.max(index, 0), copy.length), 0, section);
		return copy;
	};
	if (parentId === null) return insert(tree);
	return mapSection(tree, parentId, (p) => ({
		...p,
		children: insert(p.children),
	}));
}

/** A place in the tree: an index in the children of parentId (null: the top level). */
export type Slot = { parentId: string | null; index: number };

/** Where a section is now, as a slot. */
export function currentSlot(tree: Section[], id: string): Slot | undefined {
	const parentId = parentOf(tree, id);
	if (parentId === undefined) return undefined;
	const list =
		parentId === null ? tree : (findSection(tree, parentId)?.children ?? []);
	return { parentId, index: list.findIndex((s) => s.id === id) };
}

/**
 * Every place section id could be moved to, in reading order: before each
 * section, inside it, and at the end of each list. Indexes count without the
 * moved section, as `move` expects them. Places that are too deep are included,
 * so moving there can be refused with a message.
 */
export function keyboardSlots(tree: Section[], id: string): Slot[] {
	const [rest] = removeSection(tree, id);
	const slots: Slot[] = [];
	const walk = (list: Section[], parentId: string | null) => {
		for (let i = 0; i <= list.length; i++) {
			slots.push({ parentId, index: i });
			if (i < list.length) walk(list[i].children, list[i].id);
		}
	};
	walk(rest, null);
	return slots;
}
