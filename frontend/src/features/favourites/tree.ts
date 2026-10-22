// Pure functions over the favourites tree: flattening for rendering, drop
// slots for drag and drop, and the optimistic move.
import type { DeleteMode, FavouriteNode } from "./api";

/** A visible row of the tree. Children of the root have depth 0. */
export type FlatItem = {
	node: FavouriteNode;
	depth: number;
	parentId: string;
};

/**
 * Where a dragged node would land: in folder `parentId` at `position`, the
 * index it has among the folder's children after the move (the server's move
 * semantics). `into` marks a drop onto the folder row itself.
 */
export type Slot = { parentId: string; position: number; into?: boolean };

/** How a slot is shown: a line at the top or bottom of a row, or the row highlighted. */
export type Indicator = {
	rowId: string;
	edge: "top" | "bottom" | "inside";
	depth: number;
};

/** The droppable area below the tree, which means "end of the root folder". */
export const ROOT_END = "root-end";

export function label(node: FavouriteNode) {
	return node.title ?? node.url ?? "";
}

/** Every node's path from the root, the root named rootName. */
export function pathsOf(root: FavouriteNode, rootName: string) {
	const paths = new Map<string, string>();
	const visit = (node: FavouriteNode, path: string) => {
		paths.set(node.id, path);
		for (const child of node.children ?? []) {
			visit(child, `${path} / ${label(child)}`);
		}
	};
	visit(root, rootName);
	return paths;
}

/** The visible rows: everything except the root and the contents of collapsed folders. */
export function flatten(
	root: FavouriteNode,
	collapsed: ReadonlySet<string>,
): FlatItem[] {
	const out: FlatItem[] = [];
	const visit = (folder: FavouriteNode, depth: number) => {
		for (const child of folder.children ?? []) {
			out.push({ node: child, depth, parentId: folder.id });
			if (child.kind === "folder" && !collapsed.has(child.id)) {
				visit(child, depth + 1);
			}
		}
	};
	visit(root, 0);
	return out;
}

/**
 * The folders a delete removes: the node itself, and with mode "delete" every
 * folder below it. Links are left out.
 */
export function removedFolders(
	node: FavouriteNode,
	mode: DeleteMode | null,
): Set<string> {
	const out = new Set<string>();
	if (node.kind !== "folder") return out;
	const visit = (folder: FavouriteNode) => {
		out.add(folder.id);
		if (mode !== "delete") return;
		for (const child of folder.children ?? []) {
			if (child.kind === "folder") visit(child);
		}
	};
	visit(node);
	return out;
}

export function findNode(
	root: FavouriteNode,
	id: string,
): FavouriteNode | undefined {
	if (root.id === id) return root;
	for (const child of root.children ?? []) {
		const found = findNode(child, id);
		if (found) return found;
	}
	return undefined;
}

/** The parent folder of a node and the node's index in it. */
export function findParent(
	root: FavouriteNode,
	id: string,
): { parent: FavouriteNode; index: number } | undefined {
	const children = root.children ?? [];
	const index = children.findIndex((c) => c.id === id);
	if (index >= 0) return { parent: root, index };
	for (const child of children) {
		const found = findParent(child, id);
		if (found) return found;
	}
	return undefined;
}

/** Whether id is ancestorId or lies below it. */
export function isInSubtree(
	root: FavouriteNode,
	ancestorId: string,
	id: string,
) {
	const ancestor = findNode(root, ancestorId);
	return ancestor !== undefined && findNode(ancestor, id) !== undefined;
}

/** The children of a folder without the dragged node. */
function others(folder: FavouriteNode, draggedId: string) {
	return (folder.children ?? []).filter((c) => c.id !== draggedId);
}

function depthOf(root: FavouriteNode, id: string): number {
	if (id === root.id) return -1;
	const found = findParent(root, id);
	return found ? depthOf(root, found.parent.id) + 1 : -1;
}

/** The slot a node is in now. */
export function currentSlot(root: FavouriteNode, id: string): Slot | undefined {
	const found = findParent(root, id);
	return found && { parentId: found.parent.id, position: found.index };
}

export function sameSlot(
	a: Slot | null | undefined,
	b: Slot | null | undefined,
) {
	return (
		!!a &&
		!!b &&
		a.parentId === b.parentId &&
		a.position === b.position &&
		!!a.into === !!b.into
	);
}

/**
 * Every slot the dragged node can be moved to, top to bottom as they appear on
 * screen, for moving with the keyboard. The dragged node's own subtree is left
 * out, and collapsed folders offer a single slot at their end.
 */
export function keyboardSlots(
	root: FavouriteNode,
	collapsed: ReadonlySet<string>,
	draggedId: string,
): Slot[] {
	const slots: Slot[] = [];
	const visit = (folder: FavouriteNode) => {
		const kids = others(folder, draggedId);
		kids.forEach((kid, i) => {
			slots.push({ parentId: folder.id, position: i });
			if (kid.kind !== "folder") return;
			if (collapsed.has(kid.id)) {
				slots.push({
					parentId: kid.id,
					position: others(kid, draggedId).length,
					into: true,
				});
			} else {
				visit(kid);
			}
		});
		slots.push({ parentId: folder.id, position: kids.length });
	};
	visit(root);
	return slots;
}

/**
 * The slot for a pointer over the row overId, at fraction (0 = top, 1 =
 * bottom) of the row's height. On a folder, the upper and lower quarters mean
 * before and after it and the middle means into it (at the end). Below an
 * expanded folder that has children, "after" is its first position, since
 * that's what's directly below the row. Returns null where the node can't go.
 */
export function pointerSlot(
	root: FavouriteNode,
	collapsed: ReadonlySet<string>,
	draggedId: string,
	overId: string,
	fraction: number,
): Slot | null {
	if (overId === ROOT_END) {
		return { parentId: root.id, position: others(root, draggedId).length };
	}
	if (isInSubtree(root, draggedId, overId)) return null;
	const over = findNode(root, overId);
	const found = findParent(root, overId);
	if (!over || !found) return null;
	const index = others(found.parent, draggedId).findIndex(
		(c) => c.id === overId,
	);
	const before = { parentId: found.parent.id, position: index };
	const after = { parentId: found.parent.id, position: index + 1 };

	if (over.kind !== "folder") return fraction < 0.5 ? before : after;
	if (fraction < 0.25) return before;
	const kids = others(over, draggedId);
	if (fraction > 0.75) {
		return collapsed.has(over.id) || kids.length === 0
			? after
			: { parentId: over.id, position: 0 };
	}
	return { parentId: over.id, position: kids.length, into: true };
}

/** How to show a slot on the visible rows, or null when there's nothing to show. */
export function indicatorFor(
	root: FavouriteNode,
	collapsed: ReadonlySet<string>,
	draggedId: string,
	slot: Slot,
): Indicator | null {
	const folder = findNode(root, slot.parentId);
	if (!folder) return null;
	const depth = depthOf(root, folder.id) + 1;
	const isRoot = folder.id === root.id;
	const kids = others(folder, draggedId);
	if (!isRoot && (slot.into || collapsed.has(folder.id) || kids.length === 0)) {
		return { rowId: folder.id, edge: "inside", depth };
	}
	if (slot.position < kids.length) {
		return { rowId: kids[slot.position].id, edge: "top", depth };
	}
	let last = kids.at(-1);
	if (!last) return null;
	// The end of a folder is below the last visible row inside it.
	for (;;) {
		if (last.kind !== "folder" || collapsed.has(last.id)) break;
		const inner = others(last, draggedId).at(-1);
		if (!inner) break;
		last = inner;
	}
	return { rowId: last.id, edge: "bottom", depth };
}

/** The tree after moving id into parentId at position (as the server does it). */
export function applyMove(
	root: FavouriteNode,
	id: string,
	parentId: string,
	position: number,
): FavouriteNode {
	const moving = findNode(root, id);
	if (!moving) return root;
	const without = (n: FavouriteNode): FavouriteNode =>
		n.children
			? {
					...n,
					children: n.children.filter((c) => c.id !== id).map(without),
				}
			: n;
	const insert = (n: FavouriteNode): FavouriteNode => {
		if (n.id === parentId) {
			const children = [...(n.children ?? [])];
			const at = Math.min(Math.max(position, 0), children.length);
			children.splice(at, 0, moving);
			return { ...n, children };
		}
		return n.children ? { ...n, children: n.children.map(insert) } : n;
	};
	return insert(without(root));
}
