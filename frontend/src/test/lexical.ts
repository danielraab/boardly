import { act } from "@testing-library/react";
import {
	$createRangeSelection,
	$getRoot,
	$getSelection,
	$isRangeSelection,
	$isTextNode,
	$setSelection,
	KEY_ENTER_COMMAND,
	type LexicalEditor,
} from "lexical";

// jsdom has no layout; Lexical measures the caret after selection changes.
Range.prototype.getBoundingClientRect ??= () => new DOMRect();
Range.prototype.getClientRects ??= () =>
	Object.assign([], { item: () => null }) as unknown as DOMRectList;

// jsdom moves its DOM selection on its own when Lexical updates the DOM, and
// Lexical would take that over as the cursor. Without the listener, the
// cursor stays where Lexical put it, as it does in a browser.
const addEventListener = document.addEventListener.bind(document);
document.addEventListener = ((type: string, ...rest: never[]) => {
	if (type !== "selectionchange")
		(addEventListener as (type: string, ...args: unknown[]) => void)(
			type,
			...rest,
		);
}) as typeof document.addEventListener;

/** The Lexical editor of a content-editable element. */
export function editorOf(element: HTMLElement): LexicalEditor {
	const editor = (element as HTMLElement & { __lexicalEditor?: LexicalEditor })
		.__lexicalEditor;
	if (!editor) throw new Error("not a Lexical editor");
	return editor;
}

/**
 * Types text at the cursor (the end of the content when there is none), one
 * character per update like a keyboard does, so Markdown shortcuts run.
 * jsdom doesn't send input events to content-editable elements. "\n" is
 * Enter.
 */
export async function typeInEditor(element: HTMLElement, text: string) {
	const editor = editorOf(element);
	for (const ch of text) {
		await act(async () => {
			if (ch === "\n") {
				editor.update(
					() => {
						if (!$isRangeSelection($getSelection())) $getRoot().selectEnd();
					},
					{ discrete: true },
				);
				editor.dispatchCommand(
					KEY_ENTER_COMMAND,
					new KeyboardEvent("keydown", { key: "Enter" }),
				);
				return;
			}
			editor.update(
				() => {
					let selection = $getSelection();
					if (!$isRangeSelection(selection)) {
						$getRoot().selectEnd();
						selection = $getSelection();
					}
					if ($isRangeSelection(selection)) selection.insertText(ch);
				},
				{ discrete: true },
			);
		});
	}
}

/** Selects all text of the editor, from its first to its last text node. */
export async function selectAll(element: HTMLElement) {
	const editor = editorOf(element);
	await act(async () => {
		editor.update(
			() => {
				const first = $getRoot().getFirstDescendant();
				const last = $getRoot().getLastDescendant();
				if (!$isTextNode(first) || !$isTextNode(last)) return;
				const selection = $createRangeSelection();
				selection.anchor.set(first.getKey(), 0, "text");
				selection.focus.set(last.getKey(), last.getTextContentSize(), "text");
				$setSelection(selection);
			},
			{ discrete: true },
		);
	});
}
