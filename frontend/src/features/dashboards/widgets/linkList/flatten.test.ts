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
