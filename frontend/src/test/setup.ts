import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import i18n, { initI18n } from "#/i18n";
import { initTheme } from "#/theme";
import { mockApi } from "./api";
import { stubColorScheme } from "./colorScheme";

initI18n();

// jsdom doesn't implement scrolling; the router's scroll restoration calls it.
window.scrollTo = () => {};

let stopTheme = () => {};

beforeEach(async () => {
	localStorage.clear();
	delete document.documentElement.dataset.theme;
	stubColorScheme();
	stopTheme = initTheme();
	mockApi();
	await i18n.changeLanguage("en");
});

afterEach(() => {
	cleanup();
	stopTheme();
	vi.unstubAllGlobals();
});
