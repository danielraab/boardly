import { screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import i18n from "#/i18n";
import { setLegalLinks } from "#/test/legal";
import { setTheme } from "#/theme";
import {
	clearAnalyticsCookies,
	parentDomains,
	start,
	stopConsent,
	writeCookie,
} from "./manager";
import type { ConsentSettings } from "./settings";

const settings: ConsentSettings = {
	service: "Google Analytics",
	cookies: ["_ga", "_ga_*"],
	revision: 42,
};

let cleanups: (() => void)[] = [];
let navigate = vi.fn();

function analyticsScript() {
	const s = document.createElement("script");
	s.type = "text/plain";
	s.dataset.category = "analytics";
	s.textContent = "window.analyticsRan = true;";
	document.head.append(s);
	cleanups.push(() => {
		for (const el of document.querySelectorAll(
			'script[data-category="analytics"]',
		))
			el.remove();
	});
}

/** Whether the snippet's script is still inert. */
function inert() {
	const scripts = [
		...document.querySelectorAll('script[data-category="analytics"]'),
	];
	return (
		scripts.length > 0 &&
		scripts.every((s) => s.getAttribute("type") === "text/plain")
	);
}

function banner() {
	return screen.getByRole("region", {
		name: /Cookies and analytics|Cookies und Statistik/,
	});
}

beforeEach(() => {
	navigate = vi.fn();
	cleanups.push(setLegalLinks({ privacy: "/privacy" }));
	analyticsScript();
});

afterEach(() => {
	stopConsent();
	for (const c of cleanups) c();
	cleanups = [];
	for (const c of document.cookie.split(";")) {
		const name = c.split("=")[0]?.trim();
		if (name)
			writeCookie(`${name}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT`);
	}
});

describe("consent banner", () => {
	it("asks first and keeps the analytics script inert", async () => {
		await start(settings, { navigate });
		const b = banner();
		expect(b).toHaveTextContent("Boardly uses Google Analytics");
		expect(within(b).getByRole("button", { name: "Accept" })).toBeVisible();
		expect(within(b).getByRole("button", { name: "Reject" })).toBeVisible();
		expect(within(b).getByRole("button", { name: "Settings" })).toBeVisible();
		// Accept and reject look alike, so neither is preferred.
		expect(within(b).getByRole("button", { name: "Accept" }).className).toBe(
			within(b).getByRole("button", { name: "Reject" }).className,
		);
		expect(
			within(b).getByRole("link", { name: "Privacy policy" }),
		).toHaveAttribute("href", "/privacy");
		expect(inert()).toBe(true);
	});

	it("runs the script after Accept and stores the choice", async () => {
		await start(settings, { navigate });
		await userEvent.click(
			within(banner()).getByRole("button", { name: "Accept" }),
		);
		await waitFor(() => expect(inert()).toBe(false));
		expect(document.cookie).toContain("boardly_consent=");
	});

	it("keeps the script inert after Reject", async () => {
		await start(settings, { navigate });
		await userEvent.click(
			within(banner()).getByRole("button", { name: "Reject" }),
		);
		expect(inert()).toBe(true);
		expect(document.cookie).toContain("boardly_consent=");
	});

	it("runs without asking after an acceptance in this revision, and asks again in another", async () => {
		await start(settings, { navigate });
		await userEvent.click(
			within(banner()).getByRole("button", { name: "Accept" }),
		);
		const cookie = document.cookie;

		// A new page load with the stored choice.
		stopConsentKeepingCookie(cookie);
		for (const s of document.querySelectorAll(
			'script[data-category="analytics"]',
		))
			s.remove();
		analyticsScript();
		await start(settings, { navigate });
		expect(
			screen.queryByRole("region", { name: "Cookies and analytics" }),
		).toBeNull();
		await waitFor(() => expect(inert()).toBe(false));

		stopConsentKeepingCookie(cookie);
		for (const s of document.querySelectorAll(
			'script[data-category="analytics"]',
		))
			s.remove();
		analyticsScript();
		await start({ ...settings, revision: 43 }, { navigate });
		expect(banner()).toBeVisible();
		expect(inert()).toBe(true);
	});

	it("routes the privacy link inside the app", async () => {
		await start(settings, { navigate });
		await userEvent.click(
			within(banner()).getByRole("link", { name: "Privacy policy" }),
		);
		expect(navigate).toHaveBeenCalledWith("/privacy");
	});

	it("opens an external privacy policy in a new tab", async () => {
		for (const m of document.querySelectorAll('meta[name="boardly-privacy"]'))
			m.remove();
		cleanups.push(setLegalLinks({ privacy: "https://example.at/datenschutz" }));
		await start(settings, { navigate });
		const link = within(banner()).getByRole("link", { name: "Privacy policy" });
		expect(link).toHaveAttribute("href", "https://example.at/datenschutz");
		expect(link).toHaveAttribute("target", "_blank");
		expect(link).toHaveAttribute("rel", "noopener noreferrer");
	});

	it("escapes the service name", async () => {
		await start(
			{ ...settings, service: "<img src=x onerror=alert(1)>" },
			{ navigate },
		);
		expect(banner()).toHaveTextContent("<img src=x onerror=alert(1)>");
		expect(banner().querySelector("img")).toBeNull();
	});

	it("switches language without a reload", async () => {
		await start(settings, { navigate });
		await i18n.changeLanguage("de");
		await waitFor(() =>
			expect(banner()).toHaveTextContent("Cookies und Statistik"),
		);
		expect(
			within(banner()).getByRole("button", { name: "Akzeptieren" }),
		).toBeVisible();
		expect(
			within(banner()).getByRole("button", { name: "Ablehnen" }),
		).toBeVisible();
		expect(
			within(banner()).getByRole("button", { name: "Einstellungen" }),
		).toBeVisible();
	});

	it("follows the theme's light or dark mode", async () => {
		await start(settings, { navigate });
		expect(document.documentElement).not.toHaveClass("cc--darkmode");
		setTheme("fjord");
		expect(document.documentElement).toHaveClass("cc--darkmode");
		setTheme("sand");
		expect(document.documentElement).not.toHaveClass("cc--darkmode");
	});
});

describe("withdrawing consent", () => {
	it("clears the analytics cookies and reloads", async () => {
		const reload = vi.fn();
		vi.stubGlobal("location", {
			...window.location,
			reload,
			hostname: "localhost",
			protocol: "http:",
		});
		await start(settings, { navigate });
		await userEvent.click(
			within(banner()).getByRole("button", { name: "Accept" }),
		);
		writeCookie("_ga=1; path=/");
		writeCookie("_ga_ABC123=2; path=/");
		writeCookie("other=3; path=/");

		const { showSettings } = await import("./manager");
		showSettings();
		const dialog = await screen.findByRole("dialog", {
			name: "Cookie settings",
		});
		expect(dialog).toHaveTextContent("_ga_*");
		await userEvent.click(
			within(dialog).getByRole("button", { name: "Reject all" }),
		);

		await waitFor(() => expect(reload).toHaveBeenCalled());
		expect(document.cookie).not.toMatch(/_ga/);
		expect(document.cookie).toContain("other=3");
	});
});

describe("cookie clearing helpers", () => {
	it("lists parent domains without the top-level one", () => {
		expect(parentDomains("dash.example.org")).toEqual(["example.org"]);
		expect(parentDomains("a.dash.example.org")).toEqual([
			"dash.example.org",
			"example.org",
		]);
		expect(parentDomains("localhost")).toEqual([]);
		expect(parentDomains("192.168.1.10")).toEqual([]);
		expect(parentDomains("::1")).toEqual([]);
	});

	it("deletes only matching cookies", () => {
		writeCookie("_ga=1; path=/");
		writeCookie("keep=1; path=/");
		clearAnalyticsCookies(["_ga"], "localhost");
		expect(document.cookie).not.toContain("_ga=");
		expect(document.cookie).toContain("keep=1");
	});
});

/** Simulates a page reload: stops the manager but keeps the stored choice. */
function stopConsentKeepingCookie(cookie: string) {
	stopConsent();
	const stored = cookie
		.split(";")
		.find((c) => c.trim().startsWith("boardly_consent="));
	if (stored) writeCookie(`${stored.trim()}; path=/`);
}
