import { $createCodeNode, $isCodeNode, CodeNode } from "@lexical/code";
import { $createLinkNode, $isLinkNode, LinkNode } from "@lexical/link";
import {
	$createListItemNode,
	$createListNode,
	$isListItemNode,
	$isListNode,
	ListItemNode,
	ListNode,
} from "@lexical/list";
import {
	$createHorizontalRuleNode,
	$isHorizontalRuleNode,
	HorizontalRuleNode,
} from "@lexical/react/LexicalHorizontalRuleNode";
import {
	$createHeadingNode,
	$createQuoteNode,
	$isHeadingNode,
	$isQuoteNode,
	HeadingNode,
	QuoteNode,
} from "@lexical/rich-text";
import {
	$createLineBreakNode,
	$createParagraphNode,
	$createTextNode,
	$getRoot,
	$getState,
	$isElementNode,
	$isLineBreakNode,
	$isTextNode,
	$setState,
	createState,
	type ElementNode,
	type LexicalNode,
	type TextFormatType,
} from "lexical";
import {
	$createAttachmentNode,
	$isAttachmentNode,
	AttachmentNode,
	UploadingNode,
} from "../editor/attachments";
import type {
	Block,
	Inline,
	Item,
	ListBlock,
	ListStyle,
	Mark,
	NoteDoc,
	TextRun,
} from "./types";

/**
 * The editor's node types: only the ones the document format can store, so
 * what the editor shows is what is saved.
 */
export const editorNodes = [
	HeadingNode,
	QuoteNode,
	ListNode,
	ListItemNode,
	LinkNode,
	CodeNode,
	HorizontalRuleNode,
	AttachmentNode,
	UploadingNode,
];

/** The stable ID of a task item, kept in the list item's node state. */
export const taskIdState = createState("taskId", {
	parse: (v) => (typeof v === "string" ? v : ""),
});

/** A new task ID: 12 random base64url characters, like the server's. */
export function newTaskId(): string {
	const bytes = crypto.getRandomValues(new Uint8Array(9));
	return btoa(String.fromCharCode(...bytes))
		.replaceAll("+", "-")
		.replaceAll("/", "_");
}

const markFormats: [Mark, TextFormatType][] = [
	["bold", "bold"],
	["italic", "italic"],
	["strike", "strikethrough"],
	["code", "code"],
];

const listTypes = {
	bullet: "bullet",
	ordered: "number",
	task: "check",
} as const satisfies Record<ListStyle, string>;

// --- Lexical → document ---

function textRun(node: LexicalNode): TextRun | null {
	if (!$isTextNode(node)) return null;
	const text = node.getTextContent();
	if (!text) return null;
	const marks = markFormats
		.filter(([, f]) => node.hasFormat(f))
		.map(([m]) => m);
	return marks.length ? { type: "text", text, marks } : { type: "text", text };
}

function inlines(parent: ElementNode): Inline[] {
	const out: Inline[] = [];
	for (const child of parent.getChildren()) {
		if ($isLineBreakNode(child)) out.push({ type: "break" });
		else if ($isLinkNode(child)) {
			const content = child
				.getAllTextNodes()
				.map(textRun)
				.filter((t) => t !== null);
			out.push({ type: "link", url: child.getURL(), content });
		} else if ($isTextNode(child)) {
			const run = textRun(child);
			if (run) out.push(run);
		} else if ($isElementNode(child)) {
			// An inline element the format doesn't know: keep its text.
			out.push(...inlines(child));
		}
	}
	return out;
}

/** The only child of an item that holds a nested list, or null. */
function nestedList(item: ListItemNode): ListNode | null {
	const children = item.getChildren();
	return children.length === 1 && $isListNode(children[0]) ? children[0] : null;
}

function listBlock(list: ListNode): ListBlock {
	const type = list.getListType();
	const style: ListStyle =
		type === "check" ? "task" : type === "number" ? "ordered" : "bullet";
	const items: Item[] = [];
	for (const child of list.getChildren()) {
		if (!$isListItemNode(child)) continue;
		const nested = nestedList(child);
		if (nested) {
			// A nested list belongs to the item before it.
			const previous = items.at(-1);
			if (previous && !previous.children) {
				previous.children = listBlock(nested);
				continue;
			}
			const item: Item = { content: [], children: listBlock(nested) };
			if (style === "task") Object.assign(item, taskFields(child));
			items.push(item);
			continue;
		}
		const item: Item = { content: inlines(child) };
		if (style === "task") Object.assign(item, taskFields(child));
		items.push(item);
	}
	return { type: "list", style, items };
}

function taskFields(item: ListItemNode): { id?: string; checked: boolean } {
	const id = $getState(item, taskIdState);
	return id
		? { id, checked: item.getChecked() === true }
		: { checked: item.getChecked() === true };
}

function block(node: LexicalNode): Block | null {
	if ($isHeadingNode(node)) {
		const level = Math.min(3, Number(node.getTag().slice(1))) as 1 | 2 | 3;
		return { type: "heading", level, content: inlines(node) };
	}
	if ($isQuoteNode(node)) return { type: "quote", content: inlines(node) };
	if ($isCodeNode(node)) return { type: "code", text: node.getTextContent() };
	if ($isHorizontalRuleNode(node)) return { type: "divider" };
	if ($isAttachmentNode(node))
		return { type: "attachment", id: node.getAttachmentId() };
	if ($isListNode(node)) {
		const list = listBlock(node);
		return list.items.length ? list : null;
	}
	if ($isElementNode(node))
		return { type: "paragraph", content: inlines(node) };
	return null;
}

/** The document of the current editor state; call inside read or update. */
export function $lexicalToDoc(): NoteDoc {
	const blocks = $getRoot()
		.getChildren()
		.map(block)
		.filter((b) => b !== null);
	// An editor always has a paragraph; an empty one alone is no content.
	if (
		blocks.length === 1 &&
		blocks[0].type === "paragraph" &&
		blocks[0].content.length === 0
	) {
		return { v: 1, blocks: [] };
	}
	return { v: 1, blocks };
}

// --- document → Lexical ---

function $textNode(run: TextRun) {
	const node = $createTextNode(run.text);
	for (const [mark, format] of markFormats) {
		if (run.marks?.includes(mark)) node.toggleFormat(format);
	}
	return node;
}

function $inlineNodes(content: Inline[]): LexicalNode[] {
	return content.map((i) => {
		if (i.type === "break") return $createLineBreakNode();
		if (i.type === "link")
			return $createLinkNode(i.url).append(...i.content.map($textNode));
		return $textNode(i);
	});
}

function $listNode(list: ListBlock): ListNode {
	const node = $createListNode(listTypes[list.style]);
	for (const item of list.items) {
		const checked = list.style === "task" ? item.checked === true : undefined;
		const li = $createListItemNode(checked);
		if (list.style === "task" && item.id) $setState(li, taskIdState, item.id);
		li.append(...$inlineNodes(item.content));
		node.append(li);
		if (item.children) {
			node.append($createListItemNode().append($listNode(item.children)));
		}
	}
	return node;
}

function $blockNode(b: Block): LexicalNode {
	switch (b.type) {
		case "paragraph":
			return $createParagraphNode().append(...$inlineNodes(b.content));
		case "heading":
			return $createHeadingNode(`h${b.level}`).append(
				...$inlineNodes(b.content),
			);
		case "quote":
			return $createQuoteNode().append(...$inlineNodes(b.content));
		case "code":
			return $createCodeNode().append(
				...b.text
					.split("\n")
					.flatMap((line, i) => [
						...(i ? [$createLineBreakNode()] : []),
						...(line ? [$createTextNode(line)] : []),
					]),
			);
		case "divider":
			return $createHorizontalRuleNode();
		case "list":
			return $listNode(b);
		case "attachment":
			return $createAttachmentNode(b.id);
	}
}

/** Replaces the editor's content with the document; call inside update. */
export function $docToLexical(doc: NoteDoc) {
	const root = $getRoot();
	root.clear();
	const nodes = doc.blocks.map($blockNode);
	root.append(...(nodes.length ? nodes : [$createParagraphNode()]));
}
