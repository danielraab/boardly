import {
	$createListItemNode,
	$createListNode,
	type ListItemNode,
} from "@lexical/list";
import { $dfs } from "@lexical/utils";
import { $getNodeByKey, createEditor, type LexicalEditor } from "lexical";
import { describe, expect, it } from "vitest";
import { $docToLexical, $lexicalToDoc, editorNodes } from "../document/lexical";
import type { Item, ListBlock, NoteDoc } from "../document/types";
import { isTaskItem } from "./plugins";
import { $moveTaskBlock, $taskBlocks, dropGap } from "./taskMove";

const task = (text: string, children?: Item[]): Item => ({
	id: `id-${text}`,
	checked: text.endsWith("!"),
	content: [{ type: "text", text }],
	...(children && {
		children: { type: "list", style: "task", items: children },
	}),
});

function open(items: Item[]): LexicalEditor {
	const editor = createEditor({
		nodes: editorNodes,
		onError: (e) => {
			throw e;
		},
	});
	const doc: NoteDoc = {
		v: 1,
		blocks: [{ type: "list", style: "task", items }],
	};
	editor.update(() => $docToLexical(doc), { discrete: true });
	return editor;
}

const keyOf = (text: string) =>
	$dfs()
		.find(
			({ node }) =>
				isTaskItem(node) && node.getChildren()[0]?.getTextContent() === text,
		)
		?.node.getKey() as string;

function move(editor: LexicalEditor, text: string, gap: number) {
	let moved = false;
	editor.update(
		() => {
			moved = $moveTaskBlock(keyOf(text), gap);
		},
		{ discrete: true },
	);
	return moved;
}

/** Puts an item into a nested list of its own, as indenting it does. */
function nest(editor: LexicalEditor, text: string) {
	editor.update(
		() => {
			const item = $getNodeByKey(keyOf(text)) as ListItemNode;
			const wrapper = $createListItemNode();
			const list = $createListNode("check");
			item.insertBefore(wrapper);
			wrapper.append(list);
			list.append(item);
		},
		{ discrete: true },
	);
}

function items(editor: LexicalEditor): Item[] {
	const block = editor.read(() => $lexicalToDoc()).blocks[0] as ListBlock;
	return block.items;
}

/** The items as text, with sublists in brackets. */
function outline(list: Item[]): string {
	return list
		.map((item) => {
			const text = item.content
				.map((c) => ("text" in c ? c.text : ""))
				.join("");
			return item.children ? `${text}[${outline(item.children.items)}]` : text;
		})
		.join(" ");
}

describe("moving task blocks", () => {
	it("moves an item up and down, keeping its ID and state", () => {
		const editor = open([task("milk"), task("bread"), task("eggs!")]);
		expect(move(editor, "eggs!", 1)).toBe(true);
		expect(outline(items(editor))).toBe("milk eggs! bread");
		expect(items(editor)[1]).toMatchObject({
			id: "id-eggs!",
			checked: true,
		});
		expect(move(editor, "milk", 3)).toBe(true);
		expect(outline(items(editor))).toBe("eggs! bread milk");
	});

	it("does nothing for the gaps next to the item", () => {
		const editor = open([task("milk"), task("bread")]);
		expect(move(editor, "milk", 0)).toBe(false);
		expect(move(editor, "milk", 1)).toBe(false);
		expect(outline(items(editor))).toBe("milk bread");
	});

	it("moves a sublist along with its item", () => {
		const editor = open([
			task("milk"),
			task("bread", [task("rye"), task("spelt")]),
			task("eggs"),
		]);
		expect(move(editor, "bread", 3)).toBe(true);
		expect(outline(items(editor))).toBe("milk eggs bread[rye spelt]");
		expect(move(editor, "bread", 0)).toBe(true);
		expect(outline(items(editor))).toBe("bread[rye spelt] milk eggs");
	});

	it("keeps sublists apart when items with sublists pass each other", () => {
		const editor = open([task("a", [task("a1")]), task("b", [task("b1")])]);
		expect(move(editor, "b", 0)).toBe(true);
		expect(outline(items(editor))).toBe("b[b1] a[a1]");
	});

	it("moves an item within its sublist only", () => {
		const editor = open([task("bread", [task("rye"), task("spelt")])]);
		const list = editor.read(() => $taskBlocks(keyOf("spelt")));
		expect(list?.blocks).toHaveLength(2);
		expect(move(editor, "spelt", 0)).toBe(true);
		expect(outline(items(editor))).toBe("bread[spelt rye]");
	});

	it("has no gap between an item and its sublist", () => {
		const editor = open([task("bread", [task("rye")]), task("eggs")]);
		const list = editor.read(() => $taskBlocks(keyOf("eggs")));
		expect(list?.blocks).toHaveLength(2);
		expect(list?.blocks[0]).toHaveLength(2);
	});

	it("doesn't let an item adopt a sublist at the start of the list", () => {
		const editor = open([task("rye"), task("milk"), task("eggs")]);
		nest(editor, "rye");
		const list = editor.read(() => $taskBlocks(keyOf("eggs")));
		expect(list?.firstGap).toBe(1);
		expect(move(editor, "eggs", 0)).toBe(false);
		expect(move(editor, "eggs", 1)).toBe(true);
		expect(outline(items(editor))).toBe("[rye] eggs milk");
	});

	it("moves a following sublist without an item along with the block", () => {
		const editor = open([task("p"), task("m", [task("m1")]), task("x")]);
		nest(editor, "x");
		expect(move(editor, "m", 0)).toBe(true);
		expect(outline(items(editor))).toBe("m[m1] [x] p");
	});
});

describe("dropGap", () => {
	const rects = [
		{ top: 0, bottom: 20 },
		{ top: 20, bottom: 60 },
		{ top: 60, bottom: 80 },
	];

	it("finds the gap before the first block whose middle is below", () => {
		expect(dropGap(-50, rects, 0)).toBe(0);
		expect(dropGap(9, rects, 0)).toBe(0);
		expect(dropGap(11, rects, 0)).toBe(1);
		expect(dropGap(39, rects, 0)).toBe(1);
		expect(dropGap(41, rects, 0)).toBe(2);
		expect(dropGap(71, rects, 0)).toBe(3);
		expect(dropGap(500, rects, 0)).toBe(3);
	});

	it("never goes before the first allowed gap", () => {
		expect(dropGap(-50, rects, 1)).toBe(1);
	});
});
