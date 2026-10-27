/**
 * The stored content of a note, version 1: a format of its own, not Markdown
 * and not the editor's. Every type has a Markdown equivalent (see the
 * note-document spec). Keep in sync with backend/internal/notes/doc.go.
 */
export type NoteDoc = { v: 1; blocks: Block[] };

export type Mark = "bold" | "italic" | "strike" | "code";

export type TextRun = { type: "text"; text: string; marks?: Mark[] };

export type Inline =
	| TextRun
	| { type: "link"; url: string; content: TextRun[] }
	| { type: "break" };

export type ListStyle = "bullet" | "ordered" | "task";

export type ListBlock = { type: "list"; style: ListStyle; items: Item[] };

/** A list item; id and checked are only set in task lists. */
export type Item = {
	id?: string;
	checked?: boolean;
	content: Inline[];
	children?: ListBlock;
};

export type Block =
	| { type: "paragraph"; content: Inline[] }
	| { type: "heading"; level: 1 | 2 | 3; content: Inline[] }
	| { type: "quote"; content: Inline[] }
	| { type: "code"; text: string }
	| { type: "divider" }
	| ListBlock;

export const emptyDoc = (): NoteDoc => ({ v: 1, blocks: [] });

/** The URL schemes links may have. */
export const linkSchemes = ["http:", "https:", "mailto:"];

/** Whether url is an absolute http, https or mailto URL. */
export function isLinkUrl(url: string): boolean {
	try {
		return linkSchemes.includes(new URL(url).protocol) && url.length <= 2048;
	} catch {
		return false;
	}
}
