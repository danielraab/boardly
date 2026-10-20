import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach, beforeEach } from "vitest";
import i18n, { initI18n } from "#/i18n";

initI18n();

// jsdom doesn't implement scrolling; the router's scroll restoration calls it.
window.scrollTo = () => {};

beforeEach(async () => {
	localStorage.clear();
	await i18n.changeLanguage("en");
});

afterEach(() => {
	cleanup();
});
