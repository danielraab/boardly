import { checkUrl, MAX_LINK_TITLE } from "#/features/favourites/validate";

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
