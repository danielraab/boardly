import { describe, expect, it, vi } from "vitest";
import { ApiError } from "#/lib/api";
import { mockApi } from "#/test/api";
import { searchStations } from "./radioBrowser";

const fm4 = {
	id: "s1",
	name: "FM4",
	url: "https://orf-live.example/fm4-q2a",
	logoUrl: "https://fm4.example/logo.png",
	logo: "/api/media/search/logo?u=x&s=y",
	country: "Austria",
	tags: ["alternative", "indie", "rock"],
	bitrate: 128,
	playable: true,
};

describe("the station search", () => {
	it("asks Boardly's server, and nothing else", async () => {
		const api = mockApi({ "GET /api/media/search": { body: [fm4] } });
		expect(await searchStations("fm 4")).toEqual([fm4]);
		const [call] = api.callsTo("GET", "/api/media/search");
		expect(new URLSearchParams(call.search).get("q")).toBe("fm 4");
		const urls = vi.mocked(fetch).mock.calls.map((c) => String(c[0]));
		expect(urls).toHaveLength(1);
		expect(urls[0]).toMatch(/^\/api\/media\/search\?/);
	});

	it("throws the server's error when the directory doesn't answer", async () => {
		mockApi({
			"GET /api/media/search": {
				status: 502,
				body: { error: { code: "directory_unreachable", message: "down" } },
			},
		});
		const error = await searchStations("fm4").catch((e) => e);
		expect(error).toBeInstanceOf(ApiError);
		expect(error.code).toBe("directory_unreachable");
	});
});
