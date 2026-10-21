// Client-side slug rules that mirror the server's (internal/slugs), so forms
// show problems before sending. The server stays authoritative.

export const MAX_SLUG = 32;

/**
 * First path segments the app uses or may use for its own routes. A user slug
 * can't be one of them. Kept equal to backend/internal/slugs/testdata/reserved.json.
 */
export const RESERVED_SLUGS: readonly string[] = [
	"about",
	"account",
	"admin",
	"api",
	"app",
	"assets",
	"auth",
	"dashboards",
	"docs",
	"favicon",
	"favourites",
	"health",
	"help",
	"karakeep",
	"login",
	"logout",
	"me",
	"new",
	"public",
	"register",
	"search",
	"settings",
	"share",
	"signup",
	"static",
	"tags",
	"u",
	"user",
	"users",
	"widgets",
];

const format = /^[a-z0-9-]{1,32}$/;

/** A translation key under dashboards.slug. */
export type SlugError = "invalid" | "reserved";

/** Lowercases what the user types into a slug field. */
export function normalizeSlugInput(value: string): string {
	return value.toLowerCase();
}

/** Checks a slug; user slugs (reserved: true) also can't be a reserved word. */
export function slugError(
	value: string,
	{ reserved = false }: { reserved?: boolean } = {},
): SlugError | null {
	if (!format.test(value)) return "invalid";
	return reserved && RESERVED_SLUGS.includes(value) ? "reserved" : null;
}

const transliterations: Record<string, string> = {
	ä: "ae",
	ö: "oe",
	ü: "ue",
	ß: "ss",
};

/** Derives a dashboard slug from its title, e.g. "Küche & Haus" → "kueche-haus". */
export function slugFromTitle(title: string): string {
	return title
		.toLowerCase()
		.replace(/[äöüß]/g, (c) => transliterations[c])
		.normalize("NFD")
		.replace(/\p{M}/gu, "")
		.replace(/[^a-z0-9]+/g, "-")
		.replace(/^-+|-+$/g, "")
		.slice(0, MAX_SLUG)
		.replace(/-+$/, "");
}
