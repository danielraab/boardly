import { afterEach, describe, expect, it, vi } from "vitest";
import { setConsentMeta } from "#/test/consent";
import { startConsent } from "./index";

const manager = vi.hoisted(() => ({ imported: vi.fn(), start: vi.fn() }));

vi.mock("./manager", () => {
	manager.imported();
	return { start: manager.start, showSettings: vi.fn() };
});

let cleanup = () => {};
afterEach(() => cleanup());

describe("startConsent", () => {
	it("doesn't load the consent manager with consent off", async () => {
		cleanup = setConsentMeta({ notice: "necessary" });
		await startConsent({ navigate: vi.fn() });
		expect(manager.imported).not.toHaveBeenCalled();
	});

	it("loads and starts it with consent on", async () => {
		cleanup = setConsentMeta({ service: "Matomo", revision: 3 });
		await startConsent({ navigate: vi.fn() });
		expect(manager.imported).toHaveBeenCalled();
		expect(manager.start).toHaveBeenCalledWith(
			{ service: "Matomo", cookies: [], revision: 3 },
			expect.anything(),
		);
	});
});
