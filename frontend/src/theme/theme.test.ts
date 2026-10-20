import { afterEach, describe, expect, it, vi } from "vitest";
import { stubColorScheme } from "#/test/colorScheme";
import { getTheme, initTheme, setTheme, THEME_STORAGE_KEY } from ".";

const root = document.documentElement;
let stop = () => {};

function start() {
	stop();
	stop = initTheme();
}

afterEach(() => {
	stop();
	vi.restoreAllMocks();
});

describe("system theme", () => {
	it.each([
		[false, "white"],
		[true, "black"],
	])("without a stored choice and dark OS = %s shows %s", (dark, theme) => {
		stubColorScheme(dark);
		start();
		expect(getTheme()).toBe("system");
		expect(root.dataset.theme).toBe(theme);
	});

	it("follows OS color scheme changes without reload", () => {
		const os = stubColorScheme(false);
		start();
		os.setDark(true);
		expect(root.dataset.theme).toBe("black");
		os.setDark(false);
		expect(root.dataset.theme).toBe("white");
	});

	it("ignores OS changes while an explicit theme is chosen", () => {
		const os = stubColorScheme(false);
		start();
		setTheme("sand");
		os.setDark(true);
		expect(root.dataset.theme).toBe("sand");
	});

	it("treats an unknown stored value as system", () => {
		localStorage.setItem(THEME_STORAGE_KEY, "neon");
		stubColorScheme(true);
		start();
		expect(getTheme()).toBe("system");
		expect(root.dataset.theme).toBe("black");
	});
});

describe("setTheme", () => {
	it("stores the choice and applies it", () => {
		setTheme("fjord");
		expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe("fjord");
		expect(root.dataset.theme).toBe("fjord");
	});

	it("restores the stored choice on the next start", () => {
		setTheme("dusk");
		delete root.dataset.theme;
		start();
		expect(getTheme()).toBe("dusk");
		expect(root.dataset.theme).toBe("dusk");
	});

	it("stores system as an explicit choice", () => {
		setTheme("ember");
		setTheme("system");
		expect(localStorage.getItem(THEME_STORAGE_KEY)).toBe("system");
		expect(root.dataset.theme).toBe("white");
	});

	it("still applies the theme when storage is unavailable", () => {
		vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		setTheme("sage");
		expect(root.dataset.theme).toBe("sage");
	});
});
