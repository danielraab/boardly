import { afterEach, describe, expect, it } from "vitest";
import { appTitle } from "./appTitle";

function setMeta(content: string) {
	const meta = document.createElement("meta");
	meta.name = "application-name";
	meta.content = content;
	document.head.append(meta);
}

describe("appTitle", () => {
	afterEach(() => {
		document.head.querySelector('meta[name="application-name"]')?.remove();
	});

	it("reads the application-name meta tag", () => {
		setMeta(" Boardly Staging ");
		expect(appTitle()).toBe("Boardly Staging");
	});

	it("falls back to Boardly without the tag", () => {
		expect(appTitle()).toBe("Boardly");
	});

	it("falls back to Boardly for a blank tag", () => {
		setMeta("  ");
		expect(appTitle()).toBe("Boardly");
	});
});
