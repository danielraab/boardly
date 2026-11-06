import { afterEach, describe, expect, it } from "vitest";
import { setConsentMeta } from "#/test/consent";
import {
	consentSettings,
	cookieNoticeVariant,
	cookiePattern,
} from "./settings";

let cleanup = () => {};
afterEach(() => cleanup());

describe("consent settings", () => {
	it("is off without the meta tag", () => {
		expect(consentSettings()).toBeNull();
	});

	it("reads a service without cookies", () => {
		cleanup = setConsentMeta({ service: "Matomo", revision: 7 });
		expect(consentSettings()).toEqual({
			service: "Matomo",
			cookies: [],
			revision: 7,
		});
	});

	it("reads a service with cookies", () => {
		cleanup = setConsentMeta({
			service: "Google Analytics",
			cookies: "_ga, _ga_*",
			revision: 42,
		});
		expect(consentSettings()).toEqual({
			service: "Google Analytics",
			cookies: ["_ga", "_ga_*"],
			revision: 42,
		});
	});
});

describe("cookie patterns", () => {
	it("turns * into a wildcard and anchors the name", () => {
		const p = cookiePattern("_ga_*");
		expect(p.test("_ga_ABC123")).toBe(true);
		expect(p.test("_ga")).toBe(false);
		expect(p.test("x_ga_A")).toBe(false);
	});

	it("escapes other regex characters", () => {
		const p = cookiePattern("a.b-c");
		expect(p.test("a.b-c")).toBe(true);
		expect(p.test("aXb-c")).toBe(false);
	});
});

describe("cookie notice variant", () => {
	it("is null without the meta tag, e.g. with consent on", () => {
		cleanup = setConsentMeta({ service: "Matomo", revision: 1 });
		expect(cookieNoticeVariant()).toBeNull();
	});

	it.each(["necessary", "cookieless-analytics"] as const)(
		"reads %s",
		(notice) => {
			cleanup = setConsentMeta({ notice });
			expect(cookieNoticeVariant()).toBe(notice);
		},
	);
});
