import { describe, expect, it } from "vitest";
import { ApiError, fileNameOf } from "#/lib/api";
import { apiError, mockApi } from "#/test/api";
import { exportData } from "./api";

describe("exportData", () => {
	it("requests the export without files and returns the file name", async () => {
		const mock = mockApi({
			"GET /api/me/export": {
				body: { format: "boardly" },
				headers: {
					"Content-Disposition":
						"attachment; filename=boardly-export-ann-2026-11-02.json",
				},
			},
		});
		const { blob, fileName } = await exportData(false);
		expect(mock.callsTo("GET", "/api/me/export")[0].search).toBe(
			"?files=false",
		);
		expect(fileName).toBe("boardly-export-ann-2026-11-02.json");
		expect(JSON.parse(await blob.text())).toEqual({ format: "boardly" });
	});

	it("requests the files and falls back to a file name", async () => {
		const mock = mockApi({ "GET /api/me/export": { body: {} } });
		const { fileName } = await exportData(true);
		expect(mock.callsTo("GET", "/api/me/export")[0].search).toBe("?files=true");
		expect(fileName).toBe("boardly-export.zip");
	});

	it("throws the rate limit with the retry time", async () => {
		mockApi({
			"GET /api/me/export": apiError(429, "rate_limited", {
				"Retry-After": "1200",
			}),
		});
		const error = await exportData(false).catch((e) => e);
		expect(error).toBeInstanceOf(ApiError);
		expect(error).toMatchObject({ status: 429, retryAfter: 1200 });
	});
});

describe("fileNameOf", () => {
	it("reads quoted and unquoted file names", () => {
		expect(fileNameOf('attachment; filename="a b.zip"')).toBe("a b.zip");
		expect(fileNameOf('attachment; filename="say \\"hi\\".json"')).toBe(
			'say "hi".json',
		);
		expect(fileNameOf("attachment; filename=x.json; size=3")).toBe("x.json");
		expect(fileNameOf("attachment")).toBeNull();
		expect(fileNameOf(null)).toBeNull();
	});
});
