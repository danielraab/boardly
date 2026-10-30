import { describe, expect, it } from "vitest";
import { mockApi } from "#/test/api";
import {
	attachmentKind,
	deleteAttachment,
	formatSize,
	setAttachmentKeep,
	uploadAttachment,
} from "./api";

describe("attachments API", () => {
	it("sends attachments to their endpoints", async () => {
		const { calls } = mockApi({
			"POST /api/attachments": { status: 201, body: { id: "a1" } },
			"PATCH /api/attachments/a1": { body: { id: "a1" } },
			"DELETE /api/attachments/a1": { status: 204 },
		});
		await uploadAttachment(
			new File(["%PDF"], "my report.pdf", { type: "application/pdf" }),
		);
		await setAttachmentKeep("a1", true);
		await deleteAttachment("a1");
		expect(calls.map((c) => [c.method, c.path, c.search])).toEqual([
			["POST", "/api/attachments", "?name=my%20report.pdf"],
			["PATCH", "/api/attachments/a1", ""],
			["DELETE", "/api/attachments/a1", ""],
		]);
		expect(calls[1].body).toEqual({ keep: true });
	});

	it("tells kinds and sizes", () => {
		expect(attachmentKind("image/svg+xml")).toBe("image");
		expect(attachmentKind("video/mp4")).toBe("video");
		expect(attachmentKind("application/pdf")).toBe("file");
		expect(formatSize(300 * 1024, "en")).toBe("300 KB");
		expect(formatSize(2.5 * 1024 * 1024, "en")).toBe("2.5 MB");
		expect(formatSize(12, "en")).toBe("12 B");
	});
});
