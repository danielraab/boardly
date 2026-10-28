import { describe, expect, it } from "vitest";
import { MAX_LINK_TITLE } from "#/features/favourites/validate";
import { MAX_NOTE_TITLE } from "#/features/notes/api";
import { sharedLink, sharedMode, sharedNote } from "./sharedLink";

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

describe("sharedNote", () => {
	it("uses title and text", () => {
		expect(sharedNote({ title: "Plan", text: "Meet at 8" })).toEqual({
			title: "Plan",
			text: "Meet at 8",
		});
	});

	it("adds the shared URL as a last line", () => {
		expect(
			sharedNote({ text: "Look at this", url: "https://example.org/a" }).text,
		).toBe("Look at this\nhttps://example.org/a");
		expect(sharedNote({ url: "https://example.org/a" }).text).toBe(
			"https://example.org/a",
		);
	});

	it("doesn't repeat a URL that is in the text or isn't a web URL", () => {
		expect(
			sharedNote({
				text: "Look https://example.org/a",
				url: "https://example.org/a",
			}).text,
		).toBe("Look https://example.org/a");
		expect(sharedNote({ text: "Hi", url: "javascript:alert(1)" }).text).toBe(
			"Hi",
		);
	});

	it("cuts the title to the note title limit", () => {
		const title = sharedNote({ title: "x".repeat(MAX_NOTE_TITLE + 5) }).title;
		expect([...title]).toHaveLength(MAX_NOTE_TITLE);
	});
});

describe("sharedMode", () => {
	it("is a link without a share, for the Add link shortcut", () => {
		expect(sharedMode({})).toBe("link");
	});

	it("is a note for text or a title without a URL", () => {
		expect(sharedMode({ text: "Meet at 8" })).toBe("note");
		expect(sharedMode({ title: "Plan" })).toBe("note");
	});

	it("is a link when a URL was shared", () => {
		expect(sharedMode({ text: "Read https://example.org" })).toBe("link");
		expect(sharedMode({ url: "https://example.org" })).toBe("link");
	});
});
