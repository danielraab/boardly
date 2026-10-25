import { renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { usePageTitle } from "./usePageTitle";

describe("usePageTitle", () => {
	it("prefixes the page title", () => {
		renderHook(() => usePageTitle("Settings"));
		expect(document.title).toBe("Settings - Boardly");
	});

	it("shows only the app name without a title", () => {
		renderHook(() => usePageTitle(""));
		expect(document.title).toBe("Boardly");
	});

	it("follows the title and resets on unmount", () => {
		const { rerender, unmount } = renderHook(
			({ title }) => usePageTitle(title),
			{ initialProps: { title: "Home" } },
		);
		rerender({ title: "Work" });
		expect(document.title).toBe("Work - Boardly");
		unmount();
		expect(document.title).toBe("Boardly");
	});
});

describe("usePageTitle with a custom app title", () => {
	afterEach(() => {
		document.head.querySelector('meta[name="application-name"]')?.remove();
	});

	it("uses the app title from the meta tag", () => {
		const meta = document.createElement("meta");
		meta.name = "application-name";
		meta.content = "Boardly Staging";
		document.head.append(meta);
		const { unmount } = renderHook(() => usePageTitle("Settings"));
		expect(document.title).toBe("Settings - Boardly Staging");
		unmount();
		expect(document.title).toBe("Boardly Staging");
	});
});
