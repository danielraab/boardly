/** Writes the consent meta tags the server would; returns a cleanup. */
export function setConsentMeta(tags: {
	service?: string;
	cookies?: string;
	revision?: number;
	notice?: "necessary" | "cookieless-analytics";
}) {
	const metas = (
		[
			["boardly-consent-analytics", tags.service],
			["boardly-consent-cookies", tags.cookies],
			["boardly-consent-revision", tags.revision?.toString()],
			["boardly-cookie-notice", tags.notice],
		] as const
	).flatMap(([name, content]) => {
		if (content === undefined) return [];
		const meta = document.createElement("meta");
		meta.name = name;
		meta.content = content;
		document.head.append(meta);
		return [meta];
	});
	return () => {
		for (const m of metas) m.remove();
	};
}
