import { afterEach, describe, expect, it, vi } from "vitest";
import { mockApi } from "#/test/api";
import { resetRadioBrowser, searchStations, toResult } from "./radioBrowser";

afterEach(() => {
	resetRadioBrowser();
	vi.restoreAllMocks();
});

const raw = {
	stationuuid: "s1",
	name: " FM4 ",
	url: "http://orf.example/fm4.m3u",
	url_resolved: "https://orf-live.example/fm4-q2a",
	favicon: "https://fm4.example/logo.png",
	country: "Austria",
	tags: "alternative, indie,rock,pop",
	bitrate: 128,
};

describe("radio-browser.info results", () => {
	it("use the resolved URL and keep three tags", () => {
		expect(toResult(raw)).toEqual({
			id: "s1",
			name: "FM4",
			url: "https://orf-live.example/fm4-q2a",
			logoUrl: "https://fm4.example/logo.png",
			country: "Austria",
			tags: ["alternative", "indie", "rock"],
			bitrate: 128,
			playable: true,
		});
	});

	it("are not playable without an https stream", () => {
		const r = toResult({
			...raw,
			url_resolved: "",
			url: "http://old.example:8000/stream",
			favicon: "",
			tags: "",
			bitrate: 0,
		});
		expect(r).toMatchObject({
			url: "http://old.example:8000/stream",
			playable: false,
			logoUrl: null,
			tags: [],
			bitrate: null,
		});
	});
});

describe("the station search", () => {
	it("asks a mirror from the server list for the most popular matches", async () => {
		vi.spyOn(Math, "random").mockReturnValue(0.9);
		const api = mockApi({
			"GET /json/servers": {
				body: [
					{ name: "de1.api.radio-browser.info" },
					{ name: "at1.api.radio-browser.info" },
				],
			},
			"GET /json/stations/search": { body: [raw] },
		});
		const results = await searchStations("fm4");
		expect(results.map((r) => r.name)).toEqual(["FM4"]);
		const [call] = api.callsTo("GET", "/json/stations/search");
		const params = new URLSearchParams(call.search);
		expect(Object.fromEntries(params)).toEqual({
			name: "fm4",
			limit: "30",
			order: "clickcount",
			reverse: "true",
			hidebroken: "true",
		});
		const urls = vi.mocked(fetch).mock.calls.map((c) => String(c[0]));
		expect(urls[0]).toBe("https://all.api.radio-browser.info/json/servers");
		expect(urls[1]).toMatch(
			/^https:\/\/at1\.api\.radio-browser\.info\/json\/stations\/search\?/,
		);

		// The mirror is chosen once.
		await searchStations("jazz");
		expect(api.callsTo("GET", "/json/servers")).toHaveLength(1);
	});

	it("falls back to a known mirror when the list fails", async () => {
		mockApi({
			"GET /json/servers": { status: 503 },
			"GET /json/stations/search": { body: [] },
		});
		await searchStations("fm4");
		const urls = vi.mocked(fetch).mock.calls.map((c) => String(c[0]));
		expect(urls[1]).toMatch(/^https:\/\/de1\.api\.radio-browser\.info\//);
	});

	it("throws when the directory doesn't answer", async () => {
		mockApi({
			"GET /json/servers": { body: [{ name: "de1.api.radio-browser.info" }] },
			"GET /json/stations/search": { status: 502 },
		});
		await expect(searchStations("fm4")).rejects.toThrow();
	});
});
