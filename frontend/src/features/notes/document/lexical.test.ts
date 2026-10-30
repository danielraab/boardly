import { readdirSync, readFileSync } from "node:fs";
import { join } from "node:path";
import { $createListItemNode, $createListNode } from "@lexical/list";
import {
	$createParagraphNode,
	$createTextNode,
	$getRoot,
	$setState,
	createEditor,
} from "lexical";
import { describe, expect, it } from "vitest";
import { $createAttachmentNode, UploadingNode } from "../editor/attachments";
import {
	$docToLexical,
	$lexicalToDoc,
	editorNodes,
	newTaskId,
	taskIdState,
} from "./lexical";
import type { NoteDoc } from "./types";

// The documents the server accepts as they are (shared with the Go tests).
const fixtures = "../backend/internal/notes/testdata/docs/valid";

function editor() {
	return createEditor({
		nodes: editorNodes,
		onError: (e) => {
			throw e;
		},
	});
}

function roundTrip(doc: NoteDoc): NoteDoc {
	const e = editor();
	e.update(() => $docToLexical(doc), { discrete: true });
	return e.read(() => $lexicalToDoc());
}

describe("Lexical mapping", () => {
	it.each(readdirSync(fixtures))("round-trips %s", (file) => {
		const doc = JSON.parse(readFileSync(join(fixtures, file), "utf8"));
		expect(roundTrip(doc)).toEqual(doc);
	});

	it("keeps task IDs and checked states", () => {
		const doc: NoteDoc = {
			v: 1,
			blocks: [
				{
					type: "list",
					style: "task",
					items: [
						{ id: "a1", checked: true, content: [{ type: "text", text: "x" }] },
					],
				},
			],
		};
		expect(roundTrip(doc)).toEqual(doc);
	});

	it("drops formatting the format can't store", () => {
		const e = editor();
		e.update(
			() => {
				const text = $createTextNode("u");
				text.toggleFormat("underline");
				text.toggleFormat("bold");
				const highlighted = $createTextNode("h");
				highlighted.toggleFormat("highlight");
				$getRoot()
					.clear()
					.append($createParagraphNode().append(text, highlighted));
			},
			{ discrete: true },
		);
		expect(e.read(() => $lexicalToDoc())).toEqual({
			v: 1,
			blocks: [
				{
					type: "paragraph",
					content: [
						{ type: "text", text: "u", marks: ["bold"] },
						{ type: "text", text: "h" },
					],
				},
			],
		});
	});

	it("reads task items without an ID and keeps given ones", () => {
		const e = editor();
		e.update(
			() => {
				const withId = $createListItemNode(false).append($createTextNode("a"));
				$setState(withId, taskIdState, "keep");
				$getRoot()
					.clear()
					.append(
						$createListNode("check").append(
							withId,
							$createListItemNode(true).append($createTextNode("b")),
						),
					);
			},
			{ discrete: true },
		);
		const doc = e.read(() => $lexicalToDoc());
		const list = doc.blocks[0];
		expect(list.type === "list" && list.items).toEqual([
			{ id: "keep", checked: false, content: [{ type: "text", text: "a" }] },
			{ checked: true, content: [{ type: "text", text: "b" }] },
		]);
	});

	it("leaves out files that are still uploading", () => {
		const e = editor();
		e.update(
			() => {
				$getRoot()
					.clear()
					.append(
						$createParagraphNode().append($createTextNode("a")),
						new UploadingNode("u1", "photo.jpg"),
						$createAttachmentNode("att-1"),
					);
			},
			{ discrete: true },
		);
		expect(e.read(() => $lexicalToDoc()).blocks).toEqual([
			{ type: "paragraph", content: [{ type: "text", text: "a" }] },
			{ type: "attachment", id: "att-1" },
		]);
	});

	it("maps an empty editor to an empty document", () => {
		expect(roundTrip({ v: 1, blocks: [] })).toEqual({ v: 1, blocks: [] });
	});

	it("makes 12-character base64url task IDs", () => {
		const id = newTaskId();
		expect(id).toMatch(/^[A-Za-z0-9_-]{12}$/);
		expect(newTaskId()).not.toBe(id);
	});
});
