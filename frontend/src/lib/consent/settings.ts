// The analytics consent settings and the cookie notice variant, from the meta
// tags the server writes into index.html. Neither exists in the Vite dev
// server, which serves index.html itself.

/** Analytics consent as the operator configured it. */
export type ConsentSettings = {
	/** The analytics service's name, e.g. "Google Analytics". */
	service: string;
	/** The service's cookie names; "*" matches any characters. */
	cookies: string[];
	/** Changes whenever the snippet or these settings change. */
	revision: number;
};

function meta(name: string): string | undefined {
	return (
		document
			.querySelector<HTMLMetaElement>(`meta[name="${name}"]`)
			?.content.trim() || undefined
	);
}

/** The analytics consent settings, or null when consent is off. */
export function consentSettings(): ConsentSettings | null {
	const service = meta("boardly-consent-analytics");
	if (!service) return null;
	const revision = Number(meta("boardly-consent-revision"));
	return {
		service,
		cookies: (meta("boardly-consent-cookies") ?? "")
			.split(",")
			.map((c) => c.trim())
			.filter(Boolean),
		revision: Number.isSafeInteger(revision) && revision > 0 ? revision : 1,
	};
}

/** A cookie name with "*" wildcards as an anchored pattern: `_ga_*` → /^_ga_.*$/. */
export function cookiePattern(name: string): RegExp {
	const escaped = name
		.split("*")
		.map((part) => part.replace(/[.*+?^${}()|[\]\\-]/g, "\\$&"))
		.join(".*");
	return new RegExp(`^${escaped}$`);
}

/**
 * What the cookie notice says while consent is off: "necessary" without an
 * analytics snippet, "cookieless-analytics" with one.
 */
export type CookieNoticeVariant = "necessary" | "cookieless-analytics";

/** The cookie notice to show, or null when there is none (consent is on). */
export function cookieNoticeVariant(): CookieNoticeVariant | null {
	const v = meta("boardly-cookie-notice");
	return v === "necessary" || v === "cookieless-analytics" ? v : null;
}
