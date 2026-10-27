import { describe, expect, it } from "vitest";
import type { FavouriteNode } from "#/features/favourites/api";
import { flattenLinks } from "./flatten";

const link = (title: string): FavouriteNode => ({
	id: title.toLowerCase(),
	kind: "link",
	title,
	url: `https://${title.toLowerCase()}.example.org/`,
});

// Work: Mail, Tools (GitLab, Deep (Grafana)), Wiki
const work: FavouriteNode[] = [
	link("Mail"),
	{
		id: "tools",
		kind: "folder",
		title: "Tools",
		children: [
			link("GitLab"),
			{
				id: "deep",
				kind: "folder",
				title: "Deep",
				children: [link("Grafana")],
			},
		],
	},
	link("Wiki"),
];

const titles = (nodes: FavouriteNode[], subfolders: boolean) =>
	flattenLinks(nodes, subfolders).map((l) => l.title);

describe("flattenLinks", () => {
	it("leaves out subfolders when they aren't included", () => {
		expect(titles(work, false)).toEqual(["Mail", "Wiki"]);
	});

	it("includes subfolders depth first, where they are", () => {
		expect(titles(work, true)).toEqual(["Mail", "GitLab", "Grafana", "Wiki"]);
	});

	it("keeps the link's fields", () => {
		expect(flattenLinks([link("Mail")], false)).toEqual([
			{
				id: "mail",
				title: "Mail",
				url: "https://mail.example.org/",
				iconImageId: undefined,
			},
		]);
	});
});

describe("flattenLinks with a tag and a sort", () => {
	const tagged = (
		title: string | null,
		url: string,
		createdAt: string,
		tags: string[] = [],
	): FavouriteNode => ({
		id: url,
		kind: "link",
		title,
		url,
		createdAt,
		tags,
	});
	const nodes: FavouriteNode[] = [
		tagged("zoo", "https://zoo/", "2026-03-01T00:00:00.000Z", ["monitoring"]),
		{
			id: "tools",
			kind: "folder",
			title: "Tools",
			children: [
				tagged("Ärzte", "https://aerzte/", "2026-03-03T00:00:00.000Z", [
					"monitoring",
				]),
			],
		},
		tagged(null, "https://beta/", "2026-03-02T00:00:00.000Z"),
	];
	const shown = (
		subfolders: boolean,
		tag: string | null,
		sort: "favourites" | "newest" | "alphabetical",
	) => flattenLinks(nodes, subfolders, tag, sort).map((l) => l.title ?? l.url);

	it("keeps only links with the tag", () => {
		expect(shown(true, "monitoring", "favourites")).toEqual(["zoo", "Ärzte"]);
		expect(shown(false, "monitoring", "favourites")).toEqual(["zoo"]);
		expect(shown(true, "other", "favourites")).toEqual([]);
	});

	it("sorts newest first", () => {
		expect(shown(true, null, "newest")).toEqual([
			"Ärzte",
			"https://beta/",
			"zoo",
		]);
	});

	it("sorts alphabetically by the shown text, ignoring case and accents", () => {
		expect(shown(true, null, "alphabetical")).toEqual([
			"Ärzte",
			"https://beta/",
			"zoo",
		]);
		expect(shown(false, null, "alphabetical")).toEqual([
			"https://beta/",
			"zoo",
		]);
	});
});
