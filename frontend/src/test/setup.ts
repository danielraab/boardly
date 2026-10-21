import "@testing-library/jest-dom/vitest";
import { cleanup, configure } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import i18n, { initI18n } from "#/i18n";
import { initTheme } from "#/theme";
import { mockApi } from "./api";
import { stubColorScheme } from "./colorScheme";

initI18n();

// The first render of a test file loads the whole route tree; under a full
// parallel run that can take longer than the default 1s of findBy* queries.
configure({ asyncUtilTimeout: 5000 });

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
