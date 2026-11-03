import { describe, expect, it, vi } from "vitest";
import { mockApi } from "#/test/api";
import { importData } from "./api";

describe("importData", () => {
	it("posts the file with the parts, folder and category", async () => {
		const mock = mockApi({ "POST /api/import": { body: { files: {} } } });
		const file = new File(["{}"], "boardly-export.json");
		await importData(["favourites", "notes"], "root", "cat", file, false);

		const [call] = mock.callsTo("POST", "/api/import");
		const params = new URLSearchParams(call.search);
		expect(Object.fromEntries(params)).toEqual({
			parts: "favourites,notes",
			folderId: "root",
			categoryId: "cat",
			skipDuplicates: "false",
		});
		const init = vi.mocked(fetch).mock.calls[0][1] as RequestInit;
		expect(init.body).toBe(file);
		expect(new Headers(init.headers).get("Content-Type")).toBe(
			"application/octet-stream",
		);
	});

	it("leaves out the folder and category when not given", async () => {
		const mock = mockApi({ "POST /api/import": { body: { files: {} } } });
		await importData(["dashboards"], null, null, new Blob(["x"]), true);
		const params = new URLSearchParams(
			mock.callsTo("POST", "/api/import")[0].search,
		);
		expect(params.has("folderId")).toBe(false);
		expect(params.has("categoryId")).toBe(false);
	});
});
