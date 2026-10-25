import { screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import i18n from "#/i18n";
import { mockApi, signedIn } from "#/test/api";
import { renderApp } from "#/test/renderApp";

// The server writes BOARDLY_APP_TITLE into this meta tag of index.html.
describe("custom app title", () => {
	beforeEach(() => {
		const meta = document.createElement("meta");
		meta.name = "application-name";
		meta.content = "Boardly Staging";
		document.head.append(meta);
	});

	afterEach(() => {
		document.head.querySelector('meta[name="application-name"]')?.remove();
	});

	it("shows it in the header and the landing page", async () => {
		renderApp("/");
		const home = await screen.findByRole("link", { name: "Boardly Staging" });
		expect(home).toHaveAttribute("href", "/");
		expect(
			await screen.findByRole("heading", { level: 1, name: "Boardly Staging" }),
		).toBeVisible();
		expect(
			screen.getByRole("heading", { name: "What Boardly Staging does" }),
		).toBeVisible();
	});

	it("ends the document title with it", async () => {
		mockApi({ "GET /api/me": signedIn() });
		renderApp("/settings");
		await waitFor(() =>
			expect(document.title).toBe("Settings - Boardly Staging"),
		);
	});

	it("names it in translated texts", async () => {
		mockApi({ "GET /api/me": signedIn() });
		await i18n.changeLanguage("de");
		const { user } = renderApp("/settings");
		const input = await screen.findByLabelText("Adresse");
		await user.clear(input);
		await user.type(input, "login");
		expect(screen.getByRole("alert")).toHaveTextContent(
			"Diese Adresse ist für Boardly Staging reserviert. Bitte wähle eine andere.",
		);
	});
});
