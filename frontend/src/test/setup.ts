import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach, vi } from "vitest";
import i18n, { initI18n } from "#/i18n";
import { mockApi } from "./api";

initI18n();

// jsdom doesn't implement scrolling; the router's scroll restoration calls it.
window.scrollTo = () => {};

beforeEach(async () => {
	localStorage.clear();
	mockApi();
	await i18n.changeLanguage("en");
});

afterEach(() => {
	cleanup();
	vi.unstubAllGlobals();
});
