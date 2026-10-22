import { readFileSync } from "node:fs";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { LANGUAGE_STORAGE_KEY } from "#/i18n";
import { stubColorScheme } from "#/test/colorScheme";
import { THEME_STORAGE_KEY, themes } from ".";

// The classic (non-module) script in index.html that sets the theme before first paint.
const html = readFileSync("index.html", "utf8");
const script = html.match(/<script>([\s\S]*?)<\/script>/)?.[1] ?? "";

function run() {
	new Function(script)();
	return document.documentElement.dataset.theme;
}

// The script fills in these elements of the document head.
let themeColor: HTMLMetaElement;
let manifest: HTMLLinkElement;

beforeEach(() => {
	themeColor = document.createElement("meta");
	themeColor.name = "theme-color";
	manifest = document.createElement("link");
	manifest.rel = "manifest";
	document.head.append(themeColor, manifest);
});

afterEach(() => {
	themeColor.remove();
	manifest.remove();
	vi.restoreAllMocks();
});

function constant(name: string) {
	return JSON.parse(
		script.match(new RegExp(`const ${name} = (.*?);\\n`))?.[1] ?? "",
	);
}

describe("index.html theme script", () => {
	it("uses the same theme ids, colours and storage key as the theme module", () => {
		expect(constant("ids")).toEqual(themes.map((t) => t.id));
		expect(constant("paper")).toEqual(
			Object.fromEntries(themes.map((t) => [t.id, t.paper])),
		);
		expect(script).toContain(`stored("${THEME_STORAGE_KEY}")`);
	});

	it("colours the title bar like the stored theme", () => {
		localStorage.setItem(THEME_STORAGE_KEY, "ember");
		run();
		expect(themeColor.content).toBe("#1f1b18");
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

describe("index.html manifest language", () => {
	it("uses the same storage key as i18n", () => {
		expect(script).toContain(`stored("${LANGUAGE_STORAGE_KEY}")`);
	});

	it("links the manifest of a stored language", () => {
		localStorage.setItem(LANGUAGE_STORAGE_KEY, "de");
		run();
		expect(manifest.getAttribute("href")).toBe("/manifest.de.webmanifest");
	});

	it.each([
		[["de-AT", "en"], "de"],
		[["fr", "de"], "de"],
		[["fr"], "en"],
	])(
		"picks the manifest for browser languages %j: %s",
		(preferred, language) => {
			vi.spyOn(navigator, "languages", "get").mockReturnValue(preferred);
			run();
			expect(manifest.getAttribute("href")).toBe(
				`/manifest.${language}.webmanifest`,
			);
		},
	);
});
