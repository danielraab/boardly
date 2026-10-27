import {
	$createListNode,
	$isListItemNode,
	$isListNode,
	ListItemNode,
} from "@lexical/list";
import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { $dfs, $findMatchingParent } from "@lexical/utils";
import {
	$getNodeByKey,
	$getSelection,
	$getState,
	$isRangeSelection,
	$isTextNode,
	$setState,
	COMMAND_PRIORITY_EDITOR,
	INDENT_CONTENT_COMMAND,
	KEY_TAB_COMMAND,
	type LexicalNode,
	OUTDENT_CONTENT_COMMAND,
} from "lexical";
import { useEffect } from "react";
import { newTaskId, taskIdState } from "../document/lexical";

/** Whether node is an item of a task list that holds its own content. */
function isTaskItem(node: LexicalNode): node is ListItemNode {
	if (!$isListItemNode(node)) return false;
	const parent = node.getParent();
	if (!$isListNode(parent) || parent.getListType() !== "check") return false;
	const children = node.getChildren();
	return !(children.length === 1 && $isListNode(children[0]));
}

/**
 * Gives every task item a unique ID: a new item, or one that was split off
 * or pasted with a copy of another item's ID, gets a new one. The first item
 * with an ID, in document order, keeps it.
 */
export function TaskIdPlugin() {
	const [editor] = useLexicalComposerContext();
	useEffect(
		() =>
			editor.registerNodeTransform(ListItemNode, (node) => {
				if (!isTaskItem(node)) return;
				const id = $getState(node, taskIdState);
				if (id) {
					const first = $dfs().find(
						({ node: n }) => isTaskItem(n) && $getState(n, taskIdState) === id,
					);
					if (first?.node.is(node)) return;
				}
				$setState(node.getWritable(), taskIdState, newTaskId());
			}),
		[editor],
	);
	return null;
}

/**
 * Tab and Shift+Tab indent and outdent list items. Outside lists, Tab moves
 * the focus as usual.
 */
export function ListTabPlugin() {
	const [editor] = useLexicalComposerContext();
	useEffect(
		() =>
			editor.registerCommand(
				KEY_TAB_COMMAND,
				(event) => {
					const selection = $getSelection();
					if (!$isRangeSelection(selection)) return false;
					const inList = selection
						.getNodes()
						.some((n) => $findMatchingParent(n, $isListItemNode) !== null);
					if (!inList) return false;
					event.preventDefault();
					return editor.dispatchCommand(
						event.shiftKey ? OUTDENT_CONTENT_COMMAND : INDENT_CONTENT_COMMAND,
						undefined,
					);
				},
				COMMAND_PRIORITY_EDITOR,
			),
		[editor],
	);
	return null;
}

function openOnModClick(event: MouseEvent) {
	if (!(event.ctrlKey || event.metaKey)) return;
	const link =
		event.target instanceof Element ? event.target.closest("a[href]") : null;
	if (!(link instanceof HTMLAnchorElement)) return;
	event.preventDefault();
	window.open(link.href, "_blank", "noopener,noreferrer");
}

/**
 * A click on a link with Ctrl or Cmd held opens it in a new tab; a plain
 * click only places the cursor.
 */
export function ModClickLinkPlugin() {
	const [editor] = useLexicalComposerContext();
	useEffect(
		() =>
			editor.registerRootListener((root, previous) => {
				previous?.removeEventListener("click", openOnModClick);
				root?.addEventListener("click", openOnModClick);
			}),
		[editor],
	);
	return null;
}

/** Makes the editor read-only while editable is false. */
export function EditablePlugin({ editable }: { editable: boolean }) {
	const [editor] = useLexicalComposerContext();
	useEffect(() => editor.setEditable(editable), [editor, editable]);
	return null;
}

/** "[ ] ", "[] " or "[x] " typed at the start of a list item. */
const taskPrefix = /^\[( |x|X)?\] $/;

/**
 * Turns a bullet or numbered item that starts with "[ ] " or "[x] " into a
 * task, so "- [ ] " makes a task like "[ ] " does. Lexical's own shortcuts
 * only run at the start of a top-level paragraph. Items around it stay in
 * lists of their kind.
 */
export function TaskShortcutPlugin() {
	const [editor] = useLexicalComposerContext();
	useEffect(
		() =>
			editor.registerUpdateListener(({ tags, dirtyLeaves, editorState }) => {
				if (tags.has("historic") || tags.has("collaboration")) return;
				const found = editorState.read(() => {
					const selection = $getSelection();
					if (!$isRangeSelection(selection) || !selection.isCollapsed())
						return null;
					const anchor = selection.anchor.getNode();
					if (!$isTextNode(anchor) || !dirtyLeaves.has(anchor.getKey()))
						return null;
					const item = anchor.getParent();
					if (!$isListItemNode(item) || !item.getFirstChild()?.is(anchor))
						return null;
					const list = item.getParent();
					if (!$isListNode(list) || list.getListType() === "check") return null;
					const match = anchor
						.getTextContent()
						.slice(0, selection.anchor.offset)
						.match(taskPrefix);
					return match
						? { key: anchor.getKey(), checked: match[1]?.toLowerCase() === "x" }
						: null;
				});
				if (!found) return;
				editor.update(() => {
					const text = $getNodeByKey(found.key);
					if (!$isTextNode(text)) return;
					const item = text.getParent();
					const list = item?.getParent();
					if (!$isListItemNode(item) || !$isListNode(list)) return;
					const prefix = text.getTextContent().indexOf("] ") + 2;
					text.spliceText(0, prefix, "", true);
					if (text.getTextContentSize() === 0) text.remove();
					if (list.getChildrenSize() === 1) {
						list.setListType("check");
					} else {
						const after = item.getNextSiblings();
						const tasks = $createListNode("check");
						list.insertAfter(tasks);
						tasks.append(item);
						if (after.length > 0) {
							const rest = $createListNode(list.getListType());
							tasks.insertAfter(rest);
							rest.append(...after);
						}
						if (list.getChildrenSize() === 0) list.remove();
					}
					item.setChecked(found.checked);
					item.select(0, 0);
				});
			}),
		[editor],
	);
	return null;
}
