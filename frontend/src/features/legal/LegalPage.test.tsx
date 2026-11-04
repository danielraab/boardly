import { act, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { NoteDoc } from "#/features/notes/document/types";
import i18n from "#/i18n";
import { apiError, mockApi } from "#/test/api";
import { setLegalLinks } from "#/test/legal";
import { renderApp } from "#/test/renderApp";

let cleanup = () => {};

afterEach(() => {
	cleanup();
	cleanup = () => {};
});

function doc(heading: string, text: string): NoteDoc {
	return {
		v: 1,
		blocks: [
			{ type: "heading", level: 1, content: [{ type: "text", text: heading }] },
			{
				type: "paragraph",
				content: [
					{ type: "text", text },
					{
						type: "link",
						url: "mailto:office@example.at",
						content: [{ type: "text", text: "office@example.at" }],
					},
				],
			},
		],
	};
}

describe("legal pages", () => {
	it("shows the imprint's content to a signed-out visitor", async () => {
		cleanup = setLegalLinks({ imprint: "/imprint" });
		const { calls } = mockApi({
			"GET /api/legal/imprint": {
				body: {
					page: "imprint",
					language: null,
					doc: doc("Imprint", "Ann Example, "),
				},
			},
		});
		renderApp("/imprint");
		expect(await screen.findByText("Ann Example,")).toBeVisible();
		expect(
			screen.getByRole("link", { name: "office@example.at" }),
		).toHaveAttribute("href", "mailto:office@example.at");
		expect(document.title).toBe("Imprint - Boardly");
		expect(
			calls.some(
				(c) => c.path === "/api/legal/imprint" && c.search === "?lang=en",
			),
		).toBe(true);
	});

	it("switches the content with the language", async () => {
		cleanup = setLegalLinks({ privacy: "/privacy" });
		mockApi({
			"GET /api/legal/privacy": ({ url }) => ({
				body:
					url.searchParams.get("lang") === "de"
						? {
								page: "privacy",
								language: "de",
								doc: doc("Datenschutzerklärung", "Deutsch"),
							}
						: {
								page: "privacy",
								language: null,
								doc: doc("Privacy notice", "English"),
							},
			}),
		});
		renderApp("/privacy");
		expect(await screen.findByText("English")).toBeVisible();
		expect(document.title).toBe("Privacy - Boardly");
		await act(() => i18n.changeLanguage("de"));
		expect(await screen.findByText("Deutsch")).toBeVisible();
		await waitFor(() => expect(document.title).toBe("Datenschutz - Boardly"));
	});

	it("titles the German imprint 'Impressum' whatever its headings", async () => {
		cleanup = setLegalLinks({ imprint: "/imprint" });
		await i18n.changeLanguage("de");
		mockApi({
			"GET /api/legal/imprint": {
				body: {
					page: "imprint",
					language: "de",
					doc: doc("Offenlegung", "Text"),
				},
			},
		});
		renderApp("/imprint");
		await screen.findByText("Offenlegung");
		expect(document.title).toBe("Impressum - Boardly");
	});

	it("shows the not-found page when the page isn't configured", async () => {
		const { calls } = mockApi();
		renderApp("/privacy");
		expect(
			await screen.findByRole("heading", { name: "Page not found" }),
		).toBeVisible();
		expect(calls.some((c) => c.path.startsWith("/api/legal"))).toBe(false);
	});

	it("shows the not-found page for an external imprint", async () => {
		cleanup = setLegalLinks({ imprint: "https://example.at/impressum" });
		const { calls } = mockApi();
		renderApp("/imprint");
		expect(
			await screen.findByRole("heading", { name: "Page not found" }),
		).toBeVisible();
		expect(calls.some((c) => c.path.startsWith("/api/legal"))).toBe(false);
	});

	it("shows an error and loads again on retry", async () => {
		cleanup = setLegalLinks({ imprint: "/imprint" });
		let fail = true;
		mockApi({
			"GET /api/legal/imprint": () =>
				fail
					? apiError(500, "internal")
					: {
							body: {
								page: "imprint",
								language: null,
								doc: doc("Imprint", "Loaded"),
							},
						},
		});
		const { user } = renderApp("/imprint");
		expect(
			await screen.findByRole("alert", {}, { timeout: 5000 }),
		).toHaveTextContent("This page couldn't be loaded.");
		fail = false;
		await user.click(screen.getByRole("button", { name: "Try again" }));
		expect(await screen.findByText("Loaded")).toBeVisible();
	});
});
