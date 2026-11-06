import { screen, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "#/i18n";
import { setConsentMeta } from "#/test/consent";
import { setLegalLinks } from "#/test/legal";
import { renderApp } from "#/test/renderApp";
import { COOKIE_NOTICE_STORAGE_KEY, resetCookieNotice } from "./CookieNotice";

let cleanups: (() => void)[] = [];

beforeEach(() => resetCookieNotice());

afterEach(() => {
	for (const c of cleanups) c();
	cleanups = [];
});

function setup(tags: Parameters<typeof setConsentMeta>[0], privacy?: string) {
	cleanups.push(setConsentMeta(tags));
	if (privacy) cleanups.push(setLegalLinks({ privacy }));
}

async function notice(name = "Cookie notice") {
	await screen.findByRole("link", { name: "Boardly" });
	return screen.queryByRole("region", { name });
}

describe("cookie notice", () => {
	it("says only necessary cookies are used, in German, with the privacy link", async () => {
		setup({ notice: "necessary" }, "/privacy");
		await i18n.changeLanguage("de");
		renderApp("/");
		const n = await notice("Cookie-Hinweis");
		expect(n).toHaveTextContent(
			"Boardly verwendet nur notwendige Cookies und Browser-Speicher, um dich angemeldet zu halten und deine Einstellungen zu merken.",
		);
		expect(n).not.toHaveTextContent("Nutzungsstatistiken");
		const link = within(n as HTMLElement).getByRole("link", {
			name: "Datenschutzerklärung",
		});
		expect(link).toHaveAttribute("href", "/privacy");
		expect(link).not.toHaveAttribute("target");
	});

	it("adds the cookieless statistics sentence with a snippet", async () => {
		setup({ notice: "cookieless-analytics" });
		renderApp("/");
		const n = await notice();
		expect(n).toHaveTextContent(
			"Boardly only uses necessary cookies and browser storage, to keep you signed in and remember your settings. Usage statistics are collected without cookies.",
		);
	});

	it("opens an external privacy policy in a new tab", async () => {
		setup({ notice: "necessary" }, "https://example.at/datenschutz");
		renderApp("/");
		const link = within((await notice()) as HTMLElement).getByRole("link", {
			name: "Privacy policy",
		});
		expect(link).toHaveAttribute("href", "https://example.at/datenschutz");
		expect(link).toHaveAttribute("target", "_blank");
		expect(link).toHaveAttribute("rel", "noopener noreferrer");
	});

	it("has no privacy link without a privacy policy", async () => {
		setup({ notice: "necessary" });
		renderApp("/");
		expect(
			within((await notice()) as HTMLElement).queryByRole("link"),
		).toBeNull();
	});

	it("stays closed after closing, also after a reload", async () => {
		setup({ notice: "necessary" });
		const first = renderApp("/");
		await first.user.click(
			await screen.findByRole("button", { name: "Close cookie notice" }),
		);
		expect(screen.queryByRole("region", { name: "Cookie notice" })).toBeNull();
		expect(localStorage.getItem(COOKIE_NOTICE_STORAGE_KEY)).toBe("necessary");
		first.unmount();

		resetCookieNotice();
		renderApp("/");
		expect(await notice()).toBeNull();
	});

	it("shows again when the variant changed", async () => {
		localStorage.setItem(COOKIE_NOTICE_STORAGE_KEY, "necessary");
		setup({ notice: "cookieless-analytics" });
		renderApp("/");
		expect(await notice()).toBeVisible();
	});

	it("stays hidden until a reload when storage is blocked", async () => {
		vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		cleanups.push(() => vi.restoreAllMocks());
		setup({ notice: "necessary" });
		const first = renderApp("/");
		await first.user.click(
			await screen.findByRole("button", { name: "Close cookie notice" }),
		);
		first.unmount();

		// Navigating in the app remounts the shell, but the page wasn't reloaded.
		renderApp("/login");
		expect(await notice()).toBeNull();

		resetCookieNotice();
		renderApp("/");
		expect(await notice()).toBeVisible();
	});

	it("switches language without a reload", async () => {
		setup({ notice: "necessary" });
		renderApp("/");
		expect(await notice()).toHaveTextContent("only uses necessary cookies");
		await i18n.changeLanguage("de");
		expect(
			await screen.findByRole("region", { name: "Cookie-Hinweis" }),
		).toHaveTextContent("verwendet nur notwendige Cookies");
	});

	it("isn't shown with consent on", async () => {
		setup({ service: "Matomo", revision: 1 });
		renderApp("/");
		expect(await notice()).toBeNull();
	});
});
