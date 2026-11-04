/** The legal pages an instance can have. */
export type LegalPageName = "imprint" | "privacy";

export const legalPageNames: LegalPageName[] = ["imprint", "privacy"];

/**
 * Where a legal page's links point: "page" is its own route in the web app
 * (content from the operator's files), "external" an operator's URL.
 */
export type LegalLink =
	| { kind: "page"; href: `/${LegalPageName}` }
	| { kind: "external"; href: string };

/**
 * The link to a legal page, from the boardly-imprint / boardly-privacy meta
 * tag the server writes into index.html, or undefined when the page isn't
 * configured (also in the Vite dev server, which serves index.html itself).
 */
export function legalLink(page: LegalPageName): LegalLink | undefined {
	const href = document
		.querySelector(`meta[name="boardly-${page}"]`)
		?.getAttribute("content")
		?.trim();
	if (!href) return undefined;
	if (href === `/${page}`) return { kind: "page", href: `/${page}` };
	if (/^https?:\/\//i.test(href)) return { kind: "external", href };
	return undefined;
}
