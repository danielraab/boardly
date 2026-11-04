import { afterEach, describe, expect, it } from "vitest";
import { legalLink } from "./legalLinks";

function setMeta(name: string, content: string) {
	const meta = document.createElement("meta");
	meta.name = name;
	meta.content = content;
	document.head.append(meta);
}

describe("legalLink", () => {
	afterEach(() => {
		for (const m of document.head.querySelectorAll('meta[name^="boardly-"]'))
			m.remove();
	});

	it("is undefined without the meta tag", () => {
		expect(legalLink("imprint")).toBeUndefined();
	});

	it("is the page's own route for files", () => {
		setMeta("boardly-imprint", "/imprint");
		expect(legalLink("imprint")).toEqual({ kind: "page", href: "/imprint" });
		expect(legalLink("privacy")).toBeUndefined();
	});

	it("is external for a URL", () => {
		setMeta("boardly-privacy", "https://example.at/datenschutz?a=1&b=2");
		expect(legalLink("privacy")).toEqual({
			kind: "external",
			href: "https://example.at/datenschutz?a=1&b=2",
		});
	});

	it("is undefined for blank content", () => {
		setMeta("boardly-privacy", "  ");
		expect(legalLink("privacy")).toBeUndefined();
	});

	it("ignores content that is neither the route nor an http(s) URL", () => {
		setMeta("boardly-imprint", "javascript:alert(1)");
		expect(legalLink("imprint")).toBeUndefined();
	});
});
