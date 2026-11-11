import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import type { FeedItem } from "#/features/feeds/api";
import { mockApi, type Routes, signedIn } from "#/test/api";
import { section, viewData } from "#/test/dashboards";
import { stubAudio } from "#/test/media";
import { renderApp } from "#/test/renderApp";
import type { FeedConfig } from ".";

afterEach(() => vi.restoreAllMocks());

function item(
	id: string,
	title: string,
	fields: Partial<FeedItem> = {},
): FeedItem {
	return {
		id,
		title,
		link: `https://news.example/${id}`,
		summary: "A summary",
		publishedAt: "2026-11-08T09:00:00.000Z",
		audio: null,
		feed: { title: "Heise", subscriptionId: "s1", icon: null, faviconId: null },
		readLaterAt: null,
		likedAt: null,
		...fields,
	};
}

function feedWidget(
	config: Partial<FeedConfig> = {},
	data: unknown = { items: [item("a", "News one")], missing: false },
): ContentWidget {
	return {
		id: "w1",
		kind: "feed",
		width: 2,
		config: {
			feedIds: [],
			show: "latest",
			count: 10,
			showSummary: false,
			title: null,
			size: "M",
			showIfPublic: false,
			...config,
		},
		data,
	};
}

function dashboard(
	sections: () => Widget[],
	{ isOwner = true, feeds = [] as unknown[], more = {} as Routes } = {},
) {
	return mockApi({
		"GET /api/me": isOwner
			? signedIn()
			: {
					status: 401,
					body: { error: { code: "unauthenticated", message: "" } },
				},
		"GET /api/feeds": { body: feeds },
		"GET /api/feed-items": { body: { items: [], nextCursor: null } },
		"GET /api/u/ann/home": () => ({
			body: viewData(
				sections(),
				isOwner ? {} : { isOwner: false, id: undefined },
			),
		}),
		...more,
	});
}

/** jsdom has no layout: summaries starting with "LONG" count as cut off. */
function stubCutOffSummaries() {
	vi.spyOn(HTMLElement.prototype, "scrollHeight", "get").mockImplementation(
		function (this: HTMLElement) {
			return this.textContent?.startsWith("LONG") ? 100 : 0;
		},
	);
}

const long = `LONG ${"word ".repeat(100)}`;

describe("the feed widget", () => {
	it("shows its items with flag buttons for the owner", async () => {
		dashboard(() => [section("Work", {}, feedWidget({ title: "News" }))]);
		renderApp("/ann/home");
		const link = await screen.findByRole("link", { name: "News one" });
		expect(link).toHaveAttribute("target", "_blank");
		expect(screen.getByRole("heading", { name: "News" })).toBeVisible();
		expect(screen.getByRole("button", { name: "Like" })).toBeVisible();
		expect(screen.queryByText("A summary")).not.toBeInTheDocument();
	});

	it("shows the summary when asked to", async () => {
		dashboard(() => [section("Work", {}, feedWidget({ showSummary: true }))]);
		renderApp("/ann/home");
		expect(await screen.findByText("A summary")).toBeVisible();
	});

	it("gives visitors no flag buttons", async () => {
		const visitorItem = { ...item("a", "News one") } as Partial<FeedItem>;
		delete visitorItem.readLaterAt;
		delete visitorItem.likedAt;
		dashboard(
			() => [section("Work", {}, feedWidget({}, { items: [visitorItem] }))],
			{ isOwner: false },
		);
		renderApp("/ann/home");
		expect(await screen.findByRole("link", { name: "News one" })).toBeVisible();
		expect(
			screen.queryByRole("button", { name: "Like" }),
		).not.toBeInTheDocument();
	});

	it("plays an episode, also for a visitor", async () => {
		const audio = stubAudio();
		const episode = item("e", "Episode", {
			link: null,
			audio: {
				url: "https://cdn.example/1.mp3",
				type: null,
				durationSeconds: null,
			},
		});
		dashboard(
			() => [section("Work", {}, feedWidget({}, { items: [episode] }))],
			{ isOwner: false },
		);
		const { user } = renderApp("/ann/home");
		await user.click(
			await screen.findByRole("button", { name: "Play Episode" }),
		);
		expect(audio.audio().src).toBe("https://cdn.example/1.mp3");
		fireEvent.playing(audio.audio());
		expect(
			screen.getByRole("button", { name: "Pause Episode" }),
		).toHaveAttribute("aria-pressed", "true");
	});

	it("does nothing in edit mode", async () => {
		const audio = stubAudio();
		const episode = item("e", "Episode", {
			audio: {
				url: "https://cdn.example/1.mp3",
				type: null,
				durationSeconds: null,
			},
		});
		dashboard(() => [section("Work", {}, feedWidget())], {
			more: {
				"GET /api/feed-items": {
					body: {
						items: [episode, item("n", "News two", { audio: null })],
						nextCursor: null,
					},
				},
			},
		});
		renderApp("/ann/home?edit=true");
		expect(await screen.findByText("News two")).toBeVisible();
		expect(
			screen.queryByRole("link", { name: "News two" }),
		).not.toBeInTheDocument();
		expect(screen.getByRole("button", { name: "Play Episode" })).toBeDisabled();
		expect(
			screen.queryByRole("button", { name: "Like" }),
		).not.toBeInTheDocument();
		expect(audio.play).not.toHaveBeenCalled();
	});

	it("expands a cut-off summary for a visitor", async () => {
		stubCutOffSummaries();
		const visitorItem = {
			...item("a", "News one", { summary: long }),
		} as Partial<FeedItem>;
		delete visitorItem.readLaterAt;
		delete visitorItem.likedAt;
		dashboard(
			() => [
				section(
					"Work",
					{},
					feedWidget({ showSummary: true }, { items: [visitorItem] }),
				),
			],
			{ isOwner: false },
		);
		const { user } = renderApp("/ann/home");
		await user.click(await screen.findByRole("button", { name: "Show more" }));
		expect(screen.getByRole("button", { name: "Show less" })).toHaveAttribute(
			"aria-expanded",
			"true",
		);
	});

	it("shows no summary toggle in edit mode", async () => {
		stubCutOffSummaries();
		dashboard(() => [section("Work", {}, feedWidget({ showSummary: true }))], {
			more: {
				"GET /api/feed-items": {
					body: {
						items: [item("n", "News two", { summary: long })],
						nextCursor: null,
					},
				},
			},
		});
		renderApp("/ann/home?edit=true");
		expect(await screen.findByText("News two")).toBeVisible();
		expect(
			screen.queryByRole("button", { name: "Show more" }),
		).not.toBeInTheDocument();
	});

	it("tells the owner when its feeds are gone", async () => {
		dashboard(() => [
			section(
				"Work",
				{},
				feedWidget({ feedIds: ["gone"] }, { items: [], missing: true }),
			),
		]);
		renderApp("/ann/home");
		expect(
			await screen.findByText("These feeds no longer exist"),
		).toBeVisible();
	});

	it("says so without items", async () => {
		dashboard(() => [
			section("Work", {}, feedWidget({}, { items: [], missing: false })),
		]);
		renderApp("/ann/home");
		expect(await screen.findByText("No items")).toBeVisible();
	});

	it("reloads the dashboard after a flag changes", async () => {
		const api = dashboard(() => [section("Work", {}, feedWidget())], {
			more: { "PATCH /api/feed-items/a": { body: item("a", "News one") } },
		});
		const { user } = renderApp("/ann/home");
		await user.click(await screen.findByRole("button", { name: "Like" }));
		await waitFor(() =>
			expect(
				api.calls.filter((c) => c.path === "/api/u/ann/home").length,
			).toBeGreaterThan(1),
		);
	});

	async function openSettings(feeds: unknown[]) {
		const api = dashboard(() => [section("Work")], { feeds });
		const { user } = renderApp("/ann/home?edit=true");
		await user.click(await screen.findByRole("button", { name: "Add widget" }));
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		await user.click(
			within(within(picker).getByRole("region", { name: "Feeds" })).getByRole(
				"button",
				{ name: /^Feed/ },
			),
		);
		const dialog = await screen.findByRole("dialog", { name: "Add feed" });
		return { api, user, dialog };
	}

	it("starts with its defaults and can be confirmed right away", async () => {
		const { dialog } = await openSettings([
			{ id: "s1", title: "Heise", icon: null, faviconId: null },
		]);
		expect(within(dialog).getByLabelText("All my feeds")).toBeChecked();
		expect(within(dialog).getByLabelText("Show")).toHaveValue("latest");
		expect(within(dialog).getByLabelText("Number of items")).toHaveValue(10);
		expect(within(dialog).getByLabelText("Show summary")).not.toBeChecked();
		expect(
			within(dialog).getByLabelText("Only items with audio"),
		).not.toBeChecked();
		expect(
			within(dialog).getByLabelText("Show on public dashboard"),
		).not.toBeChecked();
		expect(
			within(dialog).getByText(/also of feeds you add later/),
		).toBeVisible();
		expect(within(dialog).getByRole("button", { name: "Add" })).toBeEnabled();
	});

	it("saves the audio filter", async () => {
		let saved: { sections: Widget[] } | undefined;
		const api = dashboard(() => [section("Work")], {
			feeds: [{ id: "s1", title: "Heise", icon: null, faviconId: null }],
			more: {
				"PUT /api/dashboards/d1/layout": ({ body }) => {
					saved = body as { sections: Widget[] };
					return { body: { version: 1, sections: [] } };
				},
			},
		});
		const { user } = renderApp("/ann/home?edit=true");
		await user.click(await screen.findByRole("button", { name: "Add widget" }));
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		await user.click(
			within(within(picker).getByRole("region", { name: "Feeds" })).getByRole(
				"button",
				{ name: /^Feed/ },
			),
		);
		const dialog = await screen.findByRole("dialog", { name: "Add feed" });
		await user.click(within(dialog).getByLabelText("Only items with audio"));
		await user.click(within(dialog).getByRole("button", { name: "Add" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(saved).toBeDefined());
		expect(saved?.sections[0]).toMatchObject({
			children: [{ kind: "feed", config: { audioOnly: true } }],
		});
		expect(api.calls.some((c) => c.method === "PUT")).toBe(true);
	});

	it("previews only items with audio when filtered", async () => {
		const episode = item("e", "Episode", {
			audio: {
				url: "https://cdn.example/1.mp3",
				type: null,
				durationSeconds: null,
			},
		});
		dashboard(() => [section("Work", {}, feedWidget({ audioOnly: true }))], {
			more: {
				"GET /api/feed-items": {
					body: { items: [item("n", "News two"), episode], nextCursor: null },
				},
			},
		});
		renderApp("/ann/home?edit=true");
		expect(await screen.findByText("Episode")).toBeVisible();
		expect(screen.queryByText("News two")).not.toBeInTheDocument();
	});

	it("needs a valid number of items", async () => {
		const { dialog, user } = await openSettings([]);
		const count = within(dialog).getByLabelText("Number of items");
		await user.clear(count);
		await user.type(count, "101");
		expect(within(dialog).getByRole("button", { name: "Add" })).toBeDisabled();
	});

	it("points to the feeds page without feeds", async () => {
		const { dialog } = await openSettings([]);
		expect(
			within(dialog).getByText(/You don't have any feeds yet\./),
		).toBeVisible();
		expect(
			within(dialog).getByRole("link", { name: "Add feeds" }),
		).toHaveAttribute("href", "/feeds");
	});
});
