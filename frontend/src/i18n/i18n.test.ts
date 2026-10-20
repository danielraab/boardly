import i18next from "i18next";
import { afterEach, describe, expect, it, vi } from "vitest";
import de from "#/locales/de.json";
import en from "#/locales/en.json";
import { initI18n, LANGUAGE_STORAGE_KEY, setLanguage } from ".";

function keys(obj: object, prefix = ""): string[] {
	return Object.entries(obj).flatMap(([k, v]) =>
		typeof v === "object" && v !== null
			? keys(v, `${prefix}${k}.`)
			: [`${prefix}${k}`],
	);
}

function freshInstance(browserLanguages: string[]) {
	vi.spyOn(navigator, "languages", "get").mockReturnValue(browserLanguages);
	vi.spyOn(navigator, "language", "get").mockReturnValue(
		browserLanguages[0] ?? "",
	);
	return initI18n(i18next.createInstance());
}

afterEach(() => {
	vi.restoreAllMocks();
});

describe("catalogs", () => {
	it("en and de have exactly the same keys", () => {
		expect(keys(de).sort()).toEqual(keys(en).sort());
	});
});

describe("detection", () => {
	it("maps a regional German browser to German", () => {
		const i18n = freshInstance(["de-AT"]);
		expect(i18n.resolvedLanguage).toBe("de");
	});

	it("falls back to English for unsupported browser languages", () => {
		const i18n = freshInstance(["fr"]);
		expect(i18n.resolvedLanguage).toBe("en");
	});

	it("prefers a stored explicit choice over the browser language", () => {
		localStorage.setItem(LANGUAGE_STORAGE_KEY, "de");
		const i18n = freshInstance(["en-US"]);
		expect(i18n.resolvedLanguage).toBe("de");
	});

	it("does not store a detected language", () => {
		freshInstance(["de-AT"]);
		expect(localStorage.getItem(LANGUAGE_STORAGE_KEY)).toBeNull();
	});
});

describe("setLanguage", () => {
	it("stores the choice and updates the lang attribute", async () => {
		const i18n = freshInstance(["en"]);
		await setLanguage("de", i18n);
		expect(localStorage.getItem(LANGUAGE_STORAGE_KEY)).toBe("de");
		expect(document.documentElement.lang).toBe("de");
	});
});
