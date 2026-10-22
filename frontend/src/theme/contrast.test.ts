import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { themes } from ".";

// Read from disk: Vitest replaces CSS imports, even ?raw ones, with empty strings.
const css = readFileSync("src/styles.css", "utf8");

const tokens = [
	"paper",
	"surface",
	"ink",
	"muted",
	"line",
	"accent",
	"danger",
	"on-danger",
	"shadow",
];

// [foreground, background, minimum ratio] from the theming spec.
const pairs: [string, string, number][] = [
	["ink", "paper", 4.5],
	["ink", "surface", 4.5],
	["muted", "paper", 4.5],
	["muted", "surface", 4.5],
	["danger", "paper", 4.5],
	["danger", "surface", 4.5],
	["on-danger", "danger", 4.5],
	["accent", "paper", 3],
	["accent", "surface", 3],
];

function declarations(selector: string): Record<string, string> {
	const start = css.indexOf(`${selector} {`);
	if (start < 0) return {};
	const body = css.slice(start, css.indexOf("}", start));
	return Object.fromEntries(
		[...body.matchAll(/(--color-[\w-]+|color-scheme):\s*([^;]+);/g)].map(
			([, name, value]) => [name.replace("--color-", ""), value.trim()],
		),
	);
}

function luminance(hex: string): number {
	const [r, g, b] = [1, 3, 5].map((i) => {
		const c = Number.parseInt(hex.slice(i, i + 2), 16) / 255;
		return c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
	});
	return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrast(a: string, b: string): number {
	const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
	return (hi + 0.05) / (lo + 0.05);
}

describe.each(themes)("theme $id", ({ id, mode, paper }) => {
	const values = declarations(`[data-theme="${id}"]`);

	it("defines every token and its color scheme", () => {
		for (const token of tokens) expect(values[token], token).toBeDefined();
		expect(values["color-scheme"]).toBe(mode);
	});

	it("has the paper colour of the theme module", () => {
		expect(values.paper).toBe(paper);
	});

	it.each(pairs)("%s on %s meets %d:1", (fg, bg, min) => {
		expect(contrast(values[fg], values[bg])).toBeGreaterThanOrEqual(min);
	});
});

it("uses the White theme as the @theme defaults", () => {
	const defaults = declarations("@theme static");
	const white = declarations('[data-theme="white"]');
	for (const token of tokens) expect(defaults[token], token).toBe(white[token]);
});
