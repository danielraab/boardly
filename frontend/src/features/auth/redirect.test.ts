import { describe, expect, it } from "vitest";
import { safeRedirect } from "./redirect";

describe("safeRedirect", () => {
	it.each([
		"/",
		"/share?url=https%3A%2F%2Fexample.org",
		"/favourites",
		"/ann/home#top",
		"/loginhelp",
	])("accepts the local path %s", (path) => {
		expect(safeRedirect(path)).toBe(path);
	});

	it.each([
		"https://evil.example",
		"//evil.example",
		"/\\evil.example",
		"javascript:alert(1)",
		"share",
		"/login",
		"/login/verify",
		"/login?redirect=/",
		"",
		42,
		undefined,
	])("rejects %j", (value) => {
		expect(safeRedirect(value)).toBeUndefined();
	});
});
