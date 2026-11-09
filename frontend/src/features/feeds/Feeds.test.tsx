import {
	act,
	fireEvent,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
	apiError,
	mockApi,
	type Routes,
	signedIn,
	signedOut,
} from "#/test/api";
import { stubAudio } from "#/test/media";
import { renderApp } from "#/test/renderApp";
import type { FeedItem, FeedSubscription } from "./api";

beforeEach(() => localStorage.clear());
afterEach(() => vi.restoreAllMocks());

function subscription(
	id: string,
	title: string,
	fields: Partial<FeedSubscription> = {},
): FeedSubscription {
	return {
		id,
		feedId: `f-${id}`,
		url: `https://${id}.example/feed.xml`,
		title,
		customTitle: null,
		siteUrl: null,
		icon: null,
		faviconId: null,
		position: 0,
		lastFetchedAt: null,
		failing: false,
		...fields,
	};
}

function item(
	id: string,
	title: string,
	fields: Partial<FeedItem> = {},
): FeedItem {
	return {
		id,
		title,
		link: `https://news.example/${id}`,
		summary: null,
		publishedAt: "2026-11-08T09:00:00.000Z",
		audio: null,
		feed: { title: "Heise", subscriptionId: "s1", icon: null, faviconId: null },
		readLaterAt: null,
		likedAt: null,
		...fields,
	};
}

const episode = item("e1", "Episode one", {
	link: null,
	audio: {
		url: "https://cdn.example/1.mp3",
		type: "audio/mpeg",
		durationSeconds: 3600,
	},
	feed: {
		title: "Podcast Eins",
		subscriptionId: "s2",
		icon: null,
		faviconId: null,
	},
});

function setup(
	{
		feeds = [subscription("s1", "Heise")],
		items = {} as Record<string, FeedItem[]>,
		more = {} as Routes,
	} = {},
	path = "/feeds",
) {
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/feeds": { body: feeds },
		"GET /api/feed-items": ({ url }) => ({
			body: {
				items: items[url.searchParams.get("view") ?? "latest"] ?? [],
				nextCursor: null,
			},
		}),
		...more,
	});
	return { api, ...renderApp(path) };
}

const bar = () => screen.getByRole("region", { name: "Media player" });

describe("the feeds page", () => {
	it("sends signed-out visitors to the login", async () => {
		mockApi({ "GET /api/me": signedOut });
		const { router } = renderApp("/feeds");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
	});

	it("says so when there are no feeds", async () => {
		setup({ feeds: [] });
		expect(await screen.findByText("No feeds yet")).toBeVisible();
	});

	it("shows the items of the chosen view", async () => {
		const { user } = setup({
			items: {
				latest: [item("a", "First"), item("b", "Second")],
				liked: [item("b", "Second", { likedAt: "2026-11-08T10:00:00.000Z" })],
			},
		});
		expect(await screen.findByText("First")).toBeVisible();
		expect(screen.getByText("Second")).toBeVisible();
		await user.click(screen.getByRole("button", { name: "Liked" }));
		await waitFor(() =>
			expect(screen.queryByText("First")).not.toBeInTheDocument(),
		);
		expect(screen.getByText("Second")).toBeVisible();
		await user.click(
			within(screen.getByRole("group", { name: "View" })).getByRole("button", {
				name: "Read later",
			}),
		);
		expect(await screen.findByText("Nothing here yet")).toBeVisible();
	});

	it("loads more items page by page", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/feeds": { body: [subscription("s1", "Heise")] },
			"GET /api/feed-items": ({ url }) =>
				url.searchParams.get("cursor")
					? { body: { items: [item("b", "Second")], nextCursor: null } }
					: { body: { items: [item("a", "First")], nextCursor: "next" } },
		});
		const { user } = renderApp("/feeds");
		await user.click(await screen.findByRole("button", { name: "Load more" }));
		expect(await screen.findByText("Second")).toBeVisible();
		expect(screen.getByText("First")).toBeVisible();
		expect(
			screen.queryByRole("button", { name: "Load more" }),
		).not.toBeInTheDocument();
		expect(api.calls.some((c) => c.search.includes("cursor=next"))).toBe(true);
	});

	it("opens a news item in a new tab without access to the opener", async () => {
		setup({ items: { latest: [item("a", "First")] } });
		const link = await screen.findByRole("link", { name: "First" });
		expect(link).toHaveAttribute("href", "https://news.example/a");
		expect(link).toHaveAttribute("target", "_blank");
		expect(link).toHaveAttribute("rel", expect.stringContaining("noopener"));
	});

	it("flags an item, showing it right away", async () => {
		let answer: (() => void) | undefined;
		const { api, user } = setup({
			items: { latest: [item("a", "First")] },
			more: {
				"PATCH /api/feed-items/a": async () => {
					await new Promise<void>((resolve) => {
						answer = resolve;
					});
					return { body: item("a", "First") };
				},
			},
		});
		const flags = await screen.findByRole("group", {
			name: "Flags of First",
		});
		await user.click(
			within(flags).getByRole("button", { name: "Read later", pressed: false }),
		);
		expect(
			await screen.findByRole("button", {
				name: "Remove from read later",
				pressed: true,
			}),
		).toBeVisible();
		expect(api.calls.find((c) => c.method === "PATCH")?.body).toEqual({
			readLater: true,
		});
		answer?.();
	});

	it("plays an episode in the media player", async () => {
		const audio = stubAudio();
		const { user } = setup({ items: { latest: [episode] } });
		await user.click(
			await screen.findByRole("button", { name: "Play Episode one" }),
		);
		expect(audio.audio().src).toBe("https://cdn.example/1.mp3");
		expect(within(bar()).getByText("Episode one")).toBeVisible();
		fireEvent.playing(audio.audio());
		expect(within(bar()).getByText("Podcast Eins")).toBeVisible();
		expect(
			screen.getByRole("button", { name: "Pause Episode one" }),
		).toHaveAttribute("aria-pressed", "true");
	});

	describe("adding a feed", () => {
		it("adds it and closes the dialog", async () => {
			const { api, user } = setup({
				feeds: [],
				more: {
					"POST /api/feeds": {
						status: 201,
						body: subscription("s1", "Heise"),
					},
				},
			});
			await user.click(await screen.findByRole("button", { name: "Add feed" }));
			const dialog = await screen.findByRole("dialog", { name: "Add feed" });
			await user.type(
				within(dialog).getByLabelText("Feed or website URL"),
				"https://heise.example/",
			);
			await user.click(within(dialog).getByRole("button", { name: "Add" }));
			await waitFor(() =>
				expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
			);
			expect(api.calls.find((c) => c.method === "POST")?.body).toEqual({
				url: "https://heise.example/",
			});
		});

		it.each([
			[400, "invalid_url", "This isn't a valid http or https address."],
			[400, "feed_unreachable", "This address can't be reached."],
			[400, "feed_not_found", "No feed was found at this address."],
			[409, "already_subscribed", "You already have this feed."],
			[429, "rate_limited", "Too many feeds added. Try again in a moment."],
		])("keeps the dialog open on %i %s", async (status, code, message) => {
			const { user } = setup({
				more: { "POST /api/feeds": apiError(status, code) },
			});
			await user.click(await screen.findByRole("button", { name: "Add feed" }));
			const dialog = await screen.findByRole("dialog", { name: "Add feed" });
			const field = within(dialog).getByLabelText("Feed or website URL");
			await user.type(field, "https://x.example/");
			await user.click(within(dialog).getByRole("button", { name: "Add" }));
			expect(await within(dialog).findByRole("alert")).toHaveTextContent(
				message,
			);
			expect(field).toHaveValue("https://x.example/");
		});
	});

	describe("managing feeds", () => {
		const two = [
			subscription("s1", "Heise"),
			subscription("s2", "Podcast Eins", { position: 1, failing: true }),
		];

		it("shows a feed that can't be updated", async () => {
			const { user } = setup({ feeds: two });
			await user.click(
				await screen.findByRole("button", { name: "Manage feeds" }),
			);
			expect(screen.getByText("Can't be updated right now")).toBeVisible();
			expect(screen.getByText("s2.example")).toBeVisible();
		});

		it("says that saved items are kept when removing a feed", async () => {
			const { api, user } = setup({
				feeds: two,
				more: { "DELETE /api/feeds/s1": { status: 204 } },
			});
			await user.click(
				await screen.findByRole("button", { name: "Manage feeds" }),
			);
			await user.click(
				screen.getByRole("button", { name: "Actions for Heise" }),
			);
			await user.click(await screen.findByRole("menuitem", { name: "Remove" }));
			const dialog = await screen.findByRole("alertdialog", {
				name: "Remove feed?",
			});
			expect(dialog).toHaveTextContent(
				'Items you marked "Read later" or "Liked" are kept.',
			);
			await user.click(within(dialog).getByRole("button", { name: "Remove" }));
			await waitFor(() =>
				expect(api.calls.some((c) => c.method === "DELETE")).toBe(true),
			);
		});

		it("reorders with the keyboard", async () => {
			// jsdom has no layout; give each row a place so dnd-kit can move.
			vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(
				function (this: Element) {
					const li = this.closest("li");
					const index = li?.parentElement
						? Array.from(li.parentElement.children).indexOf(li)
						: 0;
					return DOMRect.fromRect({
						x: 0,
						y: index * 60,
						width: 600,
						height: 60,
					});
				},
			);
			const { api, user } = setup({
				feeds: two,
				more: { "PUT /api/feeds/order": { status: 204 } },
			});
			await user.click(
				await screen.findByRole("button", { name: "Manage feeds" }),
			);
			const handle = screen.getByRole("button", { name: "Drag Podcast Eins" });
			act(() => handle.focus());
			await user.keyboard(" ");
			await user.keyboard("{ArrowUp}");
			await user.keyboard(" ");
			await waitFor(() =>
				expect(api.calls.find((c) => c.method === "PUT")?.body).toEqual([
					"s2",
					"s1",
				]),
			);
		});
	});

	it("keeps podcasts off the media page", async () => {
		const api = mockApi({
			"GET /api/me": signedIn(),
			"GET /api/media": { body: [] },
		});
		renderApp("/media");
		expect(await screen.findByText("No media yet")).toBeVisible();
		expect(api.calls.some((c) => c.path.startsWith("/api/feed"))).toBe(false);
	});
});

describe("playing an episode", () => {
	function ready() {
		const audio = stubAudio();
		const app = setup({ items: { latest: [episode] } });
		const start = async () => {
			await app.user.click(
				await screen.findByRole("button", {
					name: /(Play|Pause) Episode one/,
				}),
			);
		};
		return { ...audio, ...app, start };
	}

	it("resumes in place after a pause, without loading it again", async () => {
		const { audio, load, play, start } = ready();
		await start();
		fireEvent.playing(audio());
		const loads = load.mock.calls.length;
		const plays = play.mock.calls.length;
		audio().currentTime = 600;
		await start();
		expect(within(bar()).getByText("Paused")).toBeVisible();
		await start();
		expect(load.mock.calls.length).toBe(loads);
		expect(play.mock.calls.length).toBe(plays + 1);
		expect(audio().currentTime).toBe(600);
	});

	it("shows and moves the position slider", async () => {
		const { audio, start } = ready();
		await start();
		const slider = within(bar()).getByRole("slider", { name: "Position" });
		fireEvent.change(slider, { target: { value: "1800" } });
		expect(audio().currentTime).toBe(1800);
		expect(slider).toHaveAttribute("aria-valuetext", "30:00 / 1:00:00");
	});

	it("continues where it stopped, also after a reload", async () => {
		localStorage.setItem(
			"boardly.player.positions",
			JSON.stringify([["e1", 1500]]),
		);
		const { audio, start } = ready();
		await start();
		fireEvent.loadedMetadata(audio());
		expect(audio().currentTime).toBe(1500);
	});

	it("stores the position when it is closed", async () => {
		const { audio, start, user } = ready();
		await start();
		audio().currentTime = 1200;
		await user.click(
			within(bar()).getByRole("button", { name: "Close player" }),
		);
		expect(
			JSON.parse(localStorage.getItem("boardly.player.positions") ?? "[]"),
		).toEqual([["e1", 1200]]);
	});

	it("finishes without an error and starts over from the beginning", async () => {
		const { audio, start, user } = ready();
		await start();
		audio().currentTime = 3599;
		fireEvent.ended(audio());
		expect(within(bar()).getByText("Finished")).toBeVisible();
		expect(within(bar()).queryByRole("alert")).not.toBeInTheDocument();
		expect(localStorage.getItem("boardly.player.positions")).toBe("[]");
		await user.click(within(bar()).getByRole("button", { name: "Play" }));
		expect(within(bar()).getByText("Connecting…")).toBeVisible();
	});

	it("says so when an episode can't be played", async () => {
		const { audio, start } = ready();
		await start();
		fireEvent.error(audio());
		expect(within(bar()).getByRole("alert")).toHaveTextContent(
			"This episode can't be played right now",
		);
		expect(
			within(bar()).getByRole("button", { name: "Try again" }),
		).toBeVisible();
	});

	it("still works when the browser refuses to store anything", async () => {
		vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
			throw new Error("denied");
		});
		const { audio, start } = ready();
		await start();
		expect(audio().src).toBe("https://cdn.example/1.mp3");
		fireEvent.ended(audio());
		expect(within(bar()).getByText("Finished")).toBeVisible();
	});
});
