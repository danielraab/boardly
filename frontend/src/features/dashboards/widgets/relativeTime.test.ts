import { describe, expect, it } from "vitest";
import { relativeTime } from "./relativeTime";

const now = new Date("2026-10-27T12:00:00Z");

function ago(ms: number) {
	return new Date(now.getTime() - ms);
}

const minute = 60 * 1000;
const hour = 60 * minute;
const day = 24 * hour;

describe("relativeTime", () => {
	it.each([
		[30 * 1000, "now", "jetzt"],
		[5 * minute, "5 minutes ago", "vor 5 Minuten"],
		[3 * hour, "3 hours ago", "vor 3 Stunden"],
		[30 * hour, "yesterday", "gestern"],
		[3 * day, "3 days ago", "vor 3 Tagen"],
		[45 * day, "last month", "letzten Monat"],
		[400 * day, "last year", "letztes Jahr"],
		[800 * day, "2 years ago", "vor 2 Jahren"],
	])("formats %d ms ago", (ms, en, de) => {
		expect(relativeTime(ago(ms), now, "en")).toBe(en);
		expect(relativeTime(ago(ms), now, "de")).toBe(de);
	});
});
