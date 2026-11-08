import { beforeEach, describe, expect, it } from "vitest";
import {
	forgetPosition,
	maxRememberedEpisodes,
	readPosition,
	savePosition,
} from "#/features/media/episodePositions";

beforeEach(() => localStorage.clear());

describe("remembered episode positions", () => {
	it("keeps the most recently played ones", () => {
		for (let i = 0; i <= maxRememberedEpisodes; i++)
			savePosition(`e${i}`, i + 1);
		expect(readPosition("e0")).toBe(0);
		expect(readPosition(`e${maxRememberedEpisodes}`)).toBe(
			maxRememberedEpisodes + 1,
		);
	});

	it("forgets a finished one", () => {
		savePosition("a", 10);
		forgetPosition("a");
		expect(readPosition("a")).toBe(0);
	});

	it("ignores broken data", () => {
		localStorage.setItem("boardly.player.positions", "{oops");
		expect(readPosition("a")).toBe(0);
	});
});
