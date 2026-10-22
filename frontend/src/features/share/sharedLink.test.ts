import { describe, expect, it } from "vitest";
import { MAX_LINK_TITLE } from "#/features/favourites/validate";
import { sharedLink } from "./sharedLink";

describe("sharedLink", () => {
	it("uses url and title", () => {
		expect(
			sharedLink({ title: "Docs", url: "https://example.org/docs" }),
		).toEqual({ url: "https://example.org/docs", title: "Docs" });
	});

	it("finds the URL in the text and uses the rest as title", () => {
		expect(sharedLink({ text: "Read this https://example.org/a" })).toEqual({
			url: "https://example.org/a",
			title: "Read this",
		});
	});

	it("leaves sentence punctuation after a URL out", () => {
		expect(sharedLink({ text: "Look (https://example.org/a?b=1)." }).url).toBe(
			"https://example.org/a?b=1",
		);
	});

	it("prefers url over a URL in the text and drops it from the title", () => {
		expect(
			sharedLink({
				text: "Great read https://example.org/x",
				url: "https://example.org/x",
			}),
		).toEqual({ url: "https://example.org/x", title: "Great read" });
	});

	it("falls back to the text when url isn't a web URL", () => {
		expect(
			sharedLink({ url: "javascript:alert(1)", text: "https://example.org" })
				.url,
		).toBe("https://example.org");
	});

	it("leaves the URL empty when nothing valid was shared", () => {
		expect(sharedLink({ text: "Just some words" })).toEqual({
			url: "",
			title: "Just some words",
		});
		expect(sharedLink({})).toEqual({ url: "", title: "" });
	});

	it("cuts the title to the link title limit", () => {
		const title = sharedLink({ title: "x".repeat(MAX_LINK_TITLE + 5) }).title;
		expect([...title]).toHaveLength(MAX_LINK_TITLE);
	});
});
