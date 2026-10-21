import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";
import { stubColorScheme } from "#/test/colorScheme";
import { THEME_STORAGE_KEY, themes } from ".";

// The classic (non-module) script in index.html that sets the theme before first paint.
const html = readFileSync("index.html", "utf8");
const script = html.match(/<script>([\s\S]*?)<\/script>/)?.[1] ?? "";

function run() {
	new Function(script)();
	return document.documentElement.dataset.theme;
}

describe("index.html theme script", () => {
	it("uses the same theme ids and storage key as the theme module", () => {
		const ids = JSON.parse(script.match(/const ids = (\[.*?\]);/)?.[1] ?? "");
		expect(ids).toEqual(themes.map((t) => t.id));
		expect(script).toContain(`localStorage.getItem("${THEME_STORAGE_KEY}")`);
	});

	it("applies a stored theme", () => {
		localStorage.setItem(THEME_STORAGE_KEY, "ember");
		expect(run()).toBe("ember");
	});

	it.each([
		[false, "white"],
		[true, "black"],
	])("resolves system with dark OS = %s to %s", (dark, theme) => {
		stubColorScheme(dark);
		localStorage.setItem(THEME_STORAGE_KEY, "system");
		expect(run()).toBe(theme);
	});

	it("treats an invalid stored value as system", () => {
		stubColorScheme(true);
		localStorage.setItem(THEME_STORAGE_KEY, "neon");
		expect(run()).toBe("black");
	});
});
