import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
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
