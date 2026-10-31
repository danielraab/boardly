import { $insertNodeToNearestRoot } from "@lexical/utils";
import {
	$getNodeByKey,
	$getRoot,
	$getSelection,
	$isRangeSelection,
	$setSelection,
	type LexicalEditor,
	type LexicalNode,
	type RangeSelection,
} from "lexical";
import { type RefObject, useEffect, useRef } from "react";

/**
 * The editor's last range selection. The cursor stays where it was when the
 * focus moves to a button or a dialog, so blocks can still go there.
 */
export function useLastSelection(
	editor: LexicalEditor,
): RefObject<RangeSelection | null> {
	const last = useRef<RangeSelection | null>(null);
	useEffect(
		() =>
			editor.registerUpdateListener(({ editorState }) => {
				editorState.read(() => {
					const selection = $getSelection();
					if ($isRangeSelection(selection)) last.current = selection.clone();
				});
			}),
		[editor],
	);
	return last;
}

/**
 * Inserts top-level nodes at the cursor, or at the last cursor when the
 * editor has none now, or at the end when it never had one; call inside
 * update.
 */
export function $insertBlocksAtCursor(
	nodes: LexicalNode[],
	last: RangeSelection | null,
) {
	let selection = $getSelection();
	if (
		!$isRangeSelection(selection) &&
		last &&
		$getNodeByKey(last.anchor.key) &&
		$getNodeByKey(last.focus.key)
	) {
		selection = last.clone();
		$setSelection(selection);
	}
	if ($isRangeSelection(selection)) {
		for (const node of nodes) $insertNodeToNearestRoot(node);
	} else {
		$getRoot().append(...nodes);
	}
}
