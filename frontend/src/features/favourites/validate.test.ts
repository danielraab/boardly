import { describe, expect, it } from "vitest";
import {
	checkDescription,
	fieldErrorFromCode,
	MAX_DESCRIPTION,
} from "./validate";

describe("checkDescription", () => {
	it("allows up to 2048 characters after trimming", () => {
		expect(checkDescription("")).toBeNull();
		expect(checkDescription(` ${"ä".repeat(MAX_DESCRIPTION)} `)).toBeNull();
		expect(checkDescription("ä".repeat(MAX_DESCRIPTION + 1))).toBe(
			"descriptionTooLong",
		);
	});

	it("maps the server's error code", () => {
		expect(fieldErrorFromCode("invalid_description", "link")).toBe(
			"descriptionTooLong",
		);
	});
});
