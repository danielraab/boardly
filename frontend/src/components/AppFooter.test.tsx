import { screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import i18n from "#/i18n";
import { mockApi, signedIn } from "#/test/api";
import { setLegalLinks } from "#/test/legal";
import { renderApp } from "#/test/renderApp";

let cleanup = () => {};

afterEach(() => {
	cleanup();
	cleanup = () => {};
});

async function footer() {
	await screen.findByRole("link", { name: "Boardly" });
	return screen.getByRole("contentinfo");
}

describe("app footer", () => {
	it("shows only 'Open source · GitHub' without legal pages", async () => {
		renderApp("/");
		const f = await footer();
		expect(within(f).getByText("Open source")).toBeVisible();
		expect(within(f).queryByRole("navigation")).toBeNull();
		const github = within(f).getByRole("link", {
			name: "GitHub (opens in new tab)",
		});
		expect(github).toHaveAttribute(
			"href",
			"https://github.com/danielraab/boardly",
		);
		expect(github).toHaveAttribute("target", "_blank");
		expect(github).toHaveAttribute("rel", "noopener noreferrer");
	});

	it("links both legal pages from files in the same tab, in German", async () => {
		cleanup = setLegalLinks({ imprint: "/imprint", privacy: "/privacy" });
		await i18n.changeLanguage("de");
		renderApp("/");
		const f = await footer();
		const nav = within(f).getByRole("navigation", { name: "Rechtliches" });
		const imprint = within(nav).getByRole("link", { name: "Impressum" });
		const privacy = within(nav).getByRole("link", { name: "Datenschutz" });
		expect(imprint).toHaveAttribute("href", "/imprint");
		expect(privacy).toHaveAttribute("href", "/privacy");
		expect(imprint).not.toHaveAttribute("target");
		expect(nav).toHaveTextContent("Impressum·Datenschutz");
	});

	it("opens an external privacy policy in a new tab", async () => {
		cleanup = setLegalLinks({ privacy: "https://example.at/datenschutz" });
		renderApp("/login");
		const f = await footer();
		const privacy = within(f).getByRole("link", {
			name: "Privacy (opens in new tab)",
		});
		expect(privacy).toHaveAttribute("href", "https://example.at/datenschutz");
		expect(privacy).toHaveAttribute("target", "_blank");
		expect(privacy).toHaveAttribute("rel", "noopener noreferrer");
		expect(within(f).queryByRole("link", { name: /Imprint/ })).toBeNull();
	});

	it("keeps the repository link with a custom app title", async () => {
		const meta = document.createElement("meta");
		meta.name = "application-name";
		meta.content = "Boardly Staging";
		document.head.append(meta);
		cleanup = () => meta.remove();
		renderApp("/");
		await screen.findByRole("link", { name: "Boardly Staging" });
		expect(
			within(screen.getByRole("contentinfo")).getByRole("link", {
				name: "GitHub (opens in new tab)",
			}),
		).toHaveAttribute("href", "https://github.com/danielraab/boardly");
	});

	it.each(["/", "/login", "/no/such/page", "/settings"])(
		"is at the end of %s",
		async (path) => {
			mockApi({ "GET /api/me": signedIn() });
			renderApp(path);
			const f = await footer();
			expect(f.previousElementSibling).toBe(screen.getByRole("main"));
		},
	);
});
