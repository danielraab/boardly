import type { Block, Inline, NoteDoc } from "./types";

/** The text of inline content, with line breaks as "\n". */
export function inlineText(content: Inline[]): string {
	return content
		.map((i) =>
			i.type === "text"
				? i.text
				: i.type === "link"
					? i.content.map((t) => t.text).join("")
					: "\n",
		)
		.join("");
}

/** The lines of text of a block, in reading order. */
function blockLines(block: Block): string[] {
	switch (block.type) {
		case "paragraph":
		case "heading":
		case "quote":
			return inlineText(block.content).split("\n");
		case "code":
			return block.text.split("\n");
		case "divider":
		case "attachment":
		case "location":
			return [];
		case "list":
			return block.items.flatMap((item) => [
				...inlineText(item.content).split("\n"),
				...(item.children ? blockLines(item.children) : []),
			]);
	}
}

/** The first non-empty line of text of a document, trimmed, or "". */
export function firstLine(doc: NoteDoc): string {
	for (const block of doc.blocks) {
		for (const line of blockLines(block)) {
			const trimmed = line.trim();
			if (trimmed) return trimmed;
		}
	}
	return "";
}

/** How many characters of the first line an untitled note shows. */
export const maxFirstLine = 80;

/**
 * The title shown for a note: its title, else its first line of text cut off
 * after 80 characters, else null (shown as "Untitled note").
 */
export function displayTitle(note: {
	title: string | null;
	content: NoteDoc;
}): string | null {
	if (note.title) return note.title;
	const line = firstLine(note.content);
	if (!line) return null;
	const chars = [...line];
	return chars.length > maxFirstLine
		? `${chars.slice(0, maxFirstLine).join("")}…`
		: line;
}
