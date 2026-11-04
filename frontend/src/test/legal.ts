import type { LegalPageName } from "#/lib/legalLinks";

/** Writes the boardly-<page> meta tag the server would; returns a cleanup. */
export function setLegalLinks(links: Partial<Record<LegalPageName, string>>) {
	const metas = Object.entries(links).map(([page, href]) => {
		const meta = document.createElement("meta");
		meta.name = `boardly-${page}`;
		meta.content = href;
		document.head.append(meta);
		return meta;
	});
	return () => {
		for (const m of metas) m.remove();
	};
}
