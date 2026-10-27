import { describe, expect, it } from "vitest";
import { displayTitle, firstLine } from "./text";
import type { NoteDoc } from "./types";

const doc = (...blocks: NoteDoc["blocks"]): NoteDoc => ({ v: 1, blocks });
const p = (text: string) => ({
	type: "paragraph" as const,
	content: text ? [{ type: "text" as const, text }] : [],
});

describe("displayTitle", () => {
	it("is the title when there is one", () => {
		expect(displayTitle({ title: "Groceries", content: doc(p("milk")) })).toBe(
			"Groceries",
		);
	});

	it("is the first line of text without a title", () => {
		expect(
			displayTitle({
				title: null,
				content: doc(
					{ type: "divider" },
					p(""),
					p("  call the plumber tomorrow "),
				),
			}),
		).toBe("call the plumber tomorrow");
	});

	it("stops at a line break", () => {
		const content = doc({
			type: "paragraph",
			content: [
				{ type: "text", text: "first" },
				{ type: "break" },
				{ type: "text", text: "second" },
			],
		});
		expect(displayTitle({ title: null, content })).toBe("first");
	});

	it("takes text from lists, links and code", () => {
		expect(
			firstLine(
				doc({
					type: "list",
					style: "task",
					items: [
						{
							id: "a",
							checked: false,
							content: [
								{
									type: "link",
									url: "https://example.org",
									content: [{ type: "text", text: "docs" }],
								},
							],
						},
					],
				}),
			),
		).toBe("docs");
		expect(firstLine(doc({ type: "code", text: "\nmake check" }))).toBe(
			"make check",
		);
	});

	it("cuts long lines after 80 characters", () => {
		const title = displayTitle({
			title: null,
			content: doc(p("x".repeat(100))),
		});
		expect(title).toBe(`${"x".repeat(80)}…`);
	});

	it("is null for an empty note", () => {
		expect(displayTitle({ title: null, content: doc(p("")) })).toBeNull();
	});
});
