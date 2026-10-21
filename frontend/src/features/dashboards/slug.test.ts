import { describe, expect, it } from "vitest";
import reserved from "../../../../backend/internal/slugs/testdata/reserved.json";
import {
	normalizeSlugInput,
	RESERVED_SLUGS,
	slugError,
	slugFromTitle,
} from "./slug";

describe("slugFromTitle", () => {
	it.each([
		["Küche & Haus", "kueche-haus"],
		["Straße", "strasse"],
		["Café Crème", "cafe-creme"],
		["  --Work--  ", "work"],
		["!!!", ""],
	])("turns %j into %j", (title, slug) => {
		expect(slugFromTitle(title)).toBe(slug);
	});

	it("cuts at 32 characters without a trailing hyphen", () => {
		const slug = slugFromTitle(`${"a".repeat(31)} b`);
		expect(slug).toBe("a".repeat(31));
		expect(slugFromTitle("x".repeat(40))).toHaveLength(32);
	});
});

describe("slugError", () => {
	it.each([
		"a",
		"home",
		"my-board-2",
		"-a",
		"a-",
		"-",
		"a".repeat(32),
	])("accepts %j", (slug) => {
		expect(slugError(slug)).toBeNull();
	});

	it.each(["", "a_b", "a.b", "Home", "a".repeat(33)])("rejects %j", (slug) => {
		expect(slugError(slug)).toBe("invalid");
	});

	it("rejects reserved words only for user slugs", () => {
		expect(slugError("login", { reserved: true })).toBe("reserved");
		expect(slugError("login")).toBeNull();
	});

	it("lowercases input", () => {
		expect(normalizeSlugInput("Work")).toBe("work");
	});

	it("keeps the reserved words equal to the server's", () => {
		expect(RESERVED_SLUGS).toEqual(reserved);
	});
});
