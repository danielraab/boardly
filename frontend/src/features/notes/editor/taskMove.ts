import { $isListItemNode, $isListNode, type ListItemNode } from "@lexical/list";
import { $findMatchingParent } from "@lexical/utils";
import {
	$getNodeByKey,
	$getSelection,
	$isRangeSelection,
	type NodeKey,
} from "lexical";
import { isTaskItem } from "./plugins";

/**
 * The items of a task's list as blocks, in order. A block is a task item and
 * every item holding a nested list right after it, which move together: the
 * document gives a nested list to the item before it. A gap is a place
 * between blocks: gap n is before block n, and gap blocks.length is the end.
 */
export type TaskBlocks = {
	blocks: NodeKey[][];
	/** The index of the task's own block. */
	from: number;
	/**
	 * The first gap an item may go to. It is 1 when the list starts with a
	 * nested list of no item, which an item placed before it would adopt.
	 */
	firstGap: number;
};

function holdsNestedList(item: ListItemNode): boolean {
	const children = item.getChildren();
	return children.length === 1 && $isListNode(children[0]);
}

/** The blocks of the list of the task item key, or null if it isn't one. */
export function $taskBlocks(key: NodeKey): TaskBlocks | null {
	const node = $getNodeByKey(key);
	if (!node || !isTaskItem(node)) return null;
	const blocks: NodeKey[][] = [];
	for (const child of node.getParentOrThrow().getChildren()) {
		if ($isListItemNode(child) && holdsNestedList(child) && blocks.length > 0)
			blocks[blocks.length - 1].push(child.getKey());
		else blocks.push([child.getKey()]);
	}
	const first = $getNodeByKey(blocks[0][0]);
	return {
		blocks,
		from: blocks.findIndex((block) => block[0] === key),
		firstGap: first && isTaskItem(first) ? 0 : 1,
	};
}

/**
 * Moves the block of the task item key to a gap of its list, keeping the
 * nodes themselves. Returns whether anything moved: a gap next to the block
 * is its current place.
 */
export function $moveTaskBlock(key: NodeKey, gap: number): boolean {
	const list = $taskBlocks(key);
	if (!list) return false;
	const { blocks, from, firstGap } = list;
	if (gap < firstGap || gap > blocks.length) return false;
	if (gap === from || gap === from + 1) return false;
	const nodes = blocks[from].flatMap((k) => $getNodeByKey(k) ?? []);
	if (gap < blocks.length) {
		const before = $getNodeByKey(blocks[gap][0]);
		for (const node of nodes) before?.insertBefore(node);
		return true;
	}
	let after = $getNodeByKey(blocks[gap - 1].at(-1) as NodeKey);
	for (const node of nodes) {
		after?.insertAfter(node);
		after = node;
	}
	return true;
}

/** The key of the task item holding the whole selection, or null. */
export function $selectedTaskItem(): NodeKey | null {
	const selection = $getSelection();
	if (!$isRangeSelection(selection)) return null;
	const [anchor, focus] = [selection.anchor, selection.focus].map((point) =>
		$findMatchingParent(point.getNode(), $isListItemNode),
	);
	if (!anchor || !focus || !anchor.is(focus) || !isTaskItem(anchor))
		return null;
	return anchor.getKey();
}

/**
 * The gap of a pointer at height y, given the rectangle of each block: before
 * the first block whose middle is below it.
 */
export function dropGap(
	y: number,
	rects: { top: number; bottom: number }[],
	firstGap: number,
): number {
	for (let gap = firstGap; gap < rects.length; gap++) {
		if (y < (rects[gap].top + rects[gap].bottom) / 2) return gap;
	}
	return rects.length;
}
