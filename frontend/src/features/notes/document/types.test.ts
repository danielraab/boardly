import { describe, expect, it } from "vitest";
import { formatCoordinate, formatPoint, isValidPoint } from "./types";

describe("formatCoordinate", () => {
	it.each([
		[48.2, "48.2"],
		[16.3738189, "16.373819"],
		[0, "0"],
		[-0.0000001, "0"],
		[-180, "-180"],
	])("writes %s as %s", (x, want) => {
		expect(formatCoordinate(x)).toBe(want);
	});

	it("writes a point as lat, lng", () => {
		expect(formatPoint({ lat: 48.208176, lng: 16.373819 })).toBe(
			"48.208176, 16.373819",
		);
	});
});

describe("isValidPoint", () => {
	it("accepts the edges and rejects what is beyond", () => {
		expect(isValidPoint(-90, 180)).toBe(true);
		expect(isValidPoint(90.1, 0)).toBe(false);
		expect(isValidPoint(0, -180.5)).toBe(false);
		expect(isValidPoint(Number.NaN, 0)).toBe(false);
	});
});
