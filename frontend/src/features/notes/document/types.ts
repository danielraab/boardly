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
	| ListBlock
	| AttachmentBlock
	| LocationBlock;

/** One of the owner's attachments, shown as its own block. */
export type AttachmentBlock = { type: "attachment"; id: string };

/** A point on a map, with an optional label, shown as its own block. */
export type LocationBlock = {
	type: "location";
	lat: number;
	lng: number;
	label?: string;
};

/** A location's point and label; a missing label is undefined. */
export type LocationValue = Omit<LocationBlock, "type">;

/** The longest label of a location, in characters. */
export const maxLocationLabel = 200;

/** A latitude or longitude rounded to 6 decimals, as the server stores it. */
export function roundCoordinate(x: number): number {
	const r = Math.round(x * 1e6) / 1e6;
	return r === 0 ? 0 : r; // no -0
}

/**
 * A coordinate with up to 6 decimals and no trailing zeros, the same way the
 * server writes it (see FormatCoordinate in backend/internal/notes/doc.go).
 */
export function formatCoordinate(x: number): string {
	return String(roundCoordinate(x));
}

/** A point as "<lat>, <lng>". */
export function formatPoint({ lat, lng }: { lat: number; lng: number }) {
	return `${formatCoordinate(lat)}, ${formatCoordinate(lng)}`;
}

/** Whether lat and lng are a point on Earth. */
export function isValidPoint(lat: number, lng: number): boolean {
	return (
		Number.isFinite(lat) &&
		Number.isFinite(lng) &&
		lat >= -90 &&
		lat <= 90 &&
		lng >= -180 &&
		lng <= 180
	);
}

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

/** doc with the task item id checked or unchecked. */
export function setTaskChecked(
	doc: NoteDoc,
	id: string,
	checked: boolean,
): NoteDoc {
	const inList = (list: ListBlock): ListBlock => ({
		...list,
		items: list.items.map((item) => ({
			...item,
			...(item.id === id && { checked }),
			...(item.children && { children: inList(item.children) }),
		})),
	});
	return {
		...doc,
		blocks: doc.blocks.map((b) => (b.type === "list" ? inList(b) : b)),
	};
}

/** The IDs of the attachments of a document, each once, in content order. */
export function attachmentIds(doc: NoteDoc): string[] {
	const ids: string[] = [];
	for (const b of doc.blocks) {
		if (b.type === "attachment" && !ids.includes(b.id)) ids.push(b.id);
	}
	return ids;
}
