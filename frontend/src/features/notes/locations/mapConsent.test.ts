import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	allowMaps,
	isAllowed,
	isStored,
	mapHost,
	needsConsent,
	resetMapConsent,
	withdrawMaps,
} from "./mapConsent";

const key = "boardly.maps.allowedHosts";

function setStyle(url: string | null) {
	document.querySelector('meta[name="boardly-map-style"]')?.remove();
	if (url === null) return;
	const meta = document.createElement("meta");
	meta.name = "boardly-map-style";
	meta.content = url;
	document.head.append(meta);
}

beforeEach(() => resetMapConsent());
afterEach(() => {
	setStyle(null);
	vi.restoreAllMocks();
});

describe("map consent", () => {
	it("reads the host from the style, with OpenFreeMap by default", () => {
		expect(mapHost()).toBe("tiles.openfreemap.org");
		setStyle("https://maps.example.org/style.json");
		expect(mapHost()).toBe("maps.example.org");
	});

	it("needs no consent for a style on the page's host", () => {
		setStyle(`https://${location.hostname}/maps/style.json`);
		expect(needsConsent()).toBe(false);
		expect(isAllowed(mapHost())).toBe(true);
		expect(needsConsent("tiles.openfreemap.org")).toBe(true);
	});

	it("allows for the page without storing", () => {
		expect(isAllowed("tiles.openfreemap.org")).toBe(false);
		allowMaps("tiles.openfreemap.org", false);
		expect(isAllowed("tiles.openfreemap.org")).toBe(true);
		expect(localStorage.getItem(key)).toBeNull();
		resetMapConsent();
		expect(isAllowed("tiles.openfreemap.org")).toBe(false);
	});

	it("stores consent per host", () => {
		allowMaps("tiles.openfreemap.org", true);
		resetMapConsent();
		expect(isAllowed("tiles.openfreemap.org")).toBe(true);
		expect(isAllowed("maps.example.org")).toBe(false);
		expect(JSON.parse(localStorage.getItem(key) ?? "")).toEqual([
			"tiles.openfreemap.org",
		]);
	});

	it("withdraws stored consent but keeps it for the page", () => {
		allowMaps("tiles.openfreemap.org", true);
		withdrawMaps("tiles.openfreemap.org");
		expect(isStored("tiles.openfreemap.org")).toBe(false);
		expect(localStorage.getItem(key)).toBeNull();
		expect(isAllowed("tiles.openfreemap.org")).toBe(true);
		resetMapConsent();
		expect(isAllowed("tiles.openfreemap.org")).toBe(false);
	});

	it("still allows for the page when storage is blocked", () => {
		vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		allowMaps("tiles.openfreemap.org", true);
		expect(isAllowed("tiles.openfreemap.org")).toBe(true);
		resetMapConsent();
		expect(isAllowed("tiles.openfreemap.org")).toBe(false);
	});
});
