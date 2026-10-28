import { checkUrl, MAX_LINK_TITLE } from "#/features/favourites/validate";
import { MAX_NOTE_TITLE } from "#/features/notes/api";

/** The query parameters the manifest's share target sends to /share. */
export type ShareParams = { title?: string; text?: string; url?: string };

// Sentence punctuation that follows a URL in text rather than belonging to it.
const URL_IN_TEXT = /https?:\/\/[^\s<>"]*[^\s<>".,;:!?)\]'’”]/i;

/**
 * Turns a share into the link to prefill: the URL from `url`, or else the
 * first web URL in `text`; the title from `title`, or else the rest of `text`.
 * Shared values are untrusted, so the URL is empty unless it is a valid web URL.
 */
export function sharedLink({ title, text, url }: ShareParams): {
	url: string;
	title: string;
} {
	const fromText = text?.match(URL_IN_TEXT)?.[0];
	const link =
		[url?.trim(), fromText].find((u) => u && checkUrl(u) === null) ?? "";

	let name = title?.trim() ?? "";
	if (name === "" && text) {
		name = (link ? text.split(link).join(" ") : text)
			.replace(/\s+/g, " ")
			.trim();
	}
	return { url: link, title: [...name].slice(0, MAX_LINK_TITLE).join("") };
}

/**
 * Turns a share into the note to prefill: the title from `title`, and the
 * text from `text` with a valid web URL from `url` added as a last line,
 * unless the text already contains it.
 */
export function sharedNote({ title, text, url }: ShareParams): {
	title: string;
	text: string;
} {
	let body = text?.trim() ?? "";
	const link = url?.trim();
	if (link && checkUrl(link) === null && !body.includes(link)) {
		body = body ? `${body}\n${link}` : link;
	}
	return {
		title: [...(title?.trim() ?? "")].slice(0, MAX_NOTE_TITLE).join(""),
		text: body,
	};
}

/** What the share page saves. */
export type ShareMode = "link" | "note";

/**
 * The share page starts as a note when a title or text was shared without a
 * URL in it, and as a link otherwise, also when nothing was shared (the "Add
 * link" shortcut).
 */
export function sharedMode(params: ShareParams): ShareMode {
	const shared = Boolean(params.title?.trim() || params.text?.trim());
	return shared && sharedLink(params).url === "" ? "note" : "link";
}
