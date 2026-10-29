import { describe, expect, it } from "vitest";
import type { FavouriteNode } from "#/features/favourites/api";
import { recentLinks } from "./recent";

const now = new Date("2026-10-27T12:00:00Z");

function ago(ms: number) {
	return new Date(now.getTime() - ms);
}

const minute = 60 * 1000;
const hour = 60 * minute;
const day = 24 * hour;

const link = (id: string, msAgo: number): FavouriteNode => ({
	id,
	kind: "link",
	title: id,
	url: `https://example.org/${id}`,
	createdAt: ago(msAgo).toISOString(),
});

describe("recentLinks", () => {
	const nodes: FavouriteNode[] = [
		link("mail", 3 * day),
		{
			id: "tools",
			kind: "folder",
			title: "Tools",
			createdAt: now.toISOString(),
			children: [link("gitlab", day), link("grafana", 30 * day)],
		},
		link("a", 3 * day),
	];
	const ids = (links: { id: string }[]) => links.map((l) => l.id);

	it("orders the links of all subfolders newest first, then by ID descending", () => {
		expect(ids(recentLinks(nodes, 10, null, now))).toEqual([
			"gitlab",
			"mail",
			"a",
			"grafana",
		]);
	});

	it("limits by count and days", () => {
		expect(ids(recentLinks(nodes, 2, null, now))).toEqual(["gitlab", "mail"]);
		expect(ids(recentLinks(nodes, 10, 7, now))).toEqual([
			"gitlab",
			"mail",
			"a",
		]);
		expect(ids(recentLinks(nodes, 10, 1, now))).toEqual(["gitlab"]);
	});
});
