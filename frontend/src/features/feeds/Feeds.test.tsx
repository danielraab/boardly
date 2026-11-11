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

const savedListen = {
	positionSeconds: 0,
	durationSeconds: 3600,
	startedAt: "2026-11-08T09:00:00.000Z",
	updatedAt: "2026-11-08T09:00:00.000Z",
	finishedAt: null,
};

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

/** jsdom has no layout: summaries starting with "LONG" count as cut off. */
function stubCutOffSummaries() {
	vi.spyOn(HTMLElement.prototype, "scrollHeight", "get").mockImplementation(
		function (this: HTMLElement) {
			return this.textContent?.startsWith("LONG") ? 100 : 0;
		},
	);
}

const long = `LONG ${"word ".repeat(100)}`;

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

	it("expands a cut-off summary without opening the link", async () => {
		stubCutOffSummaries();
		const { user, router } = setup({
			items: {
				latest: [
					item("a", "First", { summary: long }),
					item("b", "Second", { summary: "Short one" }),
				],
			},
		});
		const more = await screen.findByRole("button", { name: "Show more" });
		expect(screen.getAllByRole("button", { name: "Show more" })).toHaveLength(
			1,
		);
		expect(more).toHaveAttribute("aria-expanded", "false");
		const summary = document.getElementById(
			more.getAttribute("aria-controls") ?? "",
		);
		expect(summary).toHaveTextContent(long.trim());
		expect(screen.getByRole("link", { name: "First" })).not.toContainElement(
			more,
		);

		await user.click(more);
		const less = screen.getByRole("button", { name: "Show less" });
		expect(less).toHaveAttribute("aria-expanded", "true");
		expect(summary).not.toHaveClass("line-clamp-2");
		expect(router.state.location.pathname).toBe("/feeds");

		await user.click(less);
		expect(screen.getByRole("button", { name: "Show more" })).toHaveAttribute(
			"aria-expanded",
			"false",
		);
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

	describe("searching", () => {
		const all = {
			latest: [
				item("a", "Rust news"),
				item("b", "Go news"),
				{ ...episode, title: "Rust podcast" },
			],
			read_later: [item("c", "Rust later"), item("d", "Other later")],
		};
		function searchSetup() {
			return setup({
				feeds: [
					subscription("s1", "Heise"),
					subscription("s2", "Podcast Eins"),
				],
				more: {
					"GET /api/feed-items": ({ url }) => {
						const q = url.searchParams.get("q")?.toLowerCase() ?? "";
						const feedId = url.searchParams.get("feedId");
						const view = (url.searchParams.get("view") ??
							"latest") as keyof typeof all;
						return {
							body: {
								items: (all[view] ?? []).filter(
									(it) =>
										it.title.toLowerCase().includes(q) &&
										(!feedId || it.feed.subscriptionId === feedId),
								),
								nextCursor: null,
							},
						};
					},
				},
			});
		}

		it("searches the current view and keeps the search across views", async () => {
			const { api, user } = searchSetup();
			expect(await screen.findByText("Go news")).toBeVisible();
			await user.type(screen.getByLabelText("Search items"), "rust");
			await waitFor(() =>
				expect(screen.queryByText("Go news")).not.toBeInTheDocument(),
			);
			expect(screen.getByText("Rust news")).toBeVisible();
			expect(
				api
					.callsTo("GET", "/api/feed-items")
					.some(
						(c) =>
							c.search.includes("view=latest") && c.search.includes("q=rust"),
					),
			).toBe(true);

			await user.click(
				within(screen.getByRole("group", { name: "View" })).getByRole(
					"button",
					{ name: "Read later" },
				),
			);
			expect(await screen.findByText("Rust later")).toBeVisible();
			expect(screen.queryByText("Other later")).not.toBeInTheDocument();
			expect(screen.getByLabelText("Search items")).toHaveValue("rust");
		});

		it("filters by feed", async () => {
			const { api, user } = searchSetup();
			expect(await screen.findByText("Go news")).toBeVisible();
			await user.selectOptions(screen.getByLabelText("Feed"), "Podcast Eins");
			await waitFor(() =>
				expect(screen.queryByText("Go news")).not.toBeInTheDocument(),
			);
			expect(screen.getByText("Rust podcast")).toBeVisible();
			expect(
				api
					.callsTo("GET", "/api/feed-items")
					.some((c) => c.search.includes("feedId=s2")),
			).toBe(true);
		});

		it("says when nothing matches, and clears the search", async () => {
			const { user } = searchSetup();
			expect(await screen.findByText("Go news")).toBeVisible();
			await user.type(screen.getByLabelText("Search items"), "nothing");
			expect(await screen.findByText("No matching items")).toBeVisible();
			await user.click(screen.getByRole("button", { name: "Clear search" }));
			expect(screen.getByLabelText("Search items")).toHaveValue("");
			expect(await screen.findByText("Go news")).toBeVisible();
			expect(screen.getByText("Rust news")).toBeVisible();
		});
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

describe("the listen history", () => {
	const at = (position: number, fields = {}) => ({
		...savedListen,
		positionSeconds: position,
		...fields,
	});

	it("shows the listened episodes with their sub-views", async () => {
		const { api, user } = setup({
			items: {
				listened: [{ ...episode, listen: at(0) }],
				in_progress: [{ ...episode, title: "Half way", listen: at(600) }],
			},
		});
		await user.click(await screen.findByRole("button", { name: "Listened" }));
		expect(await screen.findByText("Episode one")).toBeVisible();
		const history = screen.getByRole("group", { name: "History" });
		expect(
			within(history).getByRole("button", { name: "All" }),
		).toHaveAttribute("aria-pressed", "true");

		await user.click(
			within(history).getByRole("button", { name: "In progress" }),
		);
		expect(await screen.findByText("Half way")).toBeVisible();
		await user.click(within(history).getByRole("button", { name: "Finished" }));
		expect(await screen.findByText("Nothing here yet")).toBeVisible();
		const views = api
			.callsTo("GET", "/api/feed-items")
			.map((c) => new URLSearchParams(c.search).get("view"));
		expect(views).toEqual(
			expect.arrayContaining(["listened", "in_progress", "finished"]),
		);

		await user.click(screen.getByRole("button", { name: "Latest" }));
		expect(
			screen.queryByRole("group", { name: "History" }),
		).not.toBeInTheDocument();
	});

	it("shows how much of an episode is left", async () => {
		setup({
			items: {
				latest: [
					{ ...episode, listen: at(2700) },
					{
						...episode,
						id: "e2",
						title: "Done",
						listen: at(0, { finishedAt: "2026-11-08T10:00:00.000Z" }),
					},
					{
						...episode,
						id: "e3",
						title: "Unknown length",
						audio: {
							url: "https://cdn.example/3.mp3",
							type: null,
							durationSeconds: null,
						},
						listen: at(100, { durationSeconds: null }),
					},
					{ ...episode, id: "e4", title: "Never played" },
				],
			},
		});
		const row = async (name: string) =>
			(await screen.findByText(name)).closest("li") as HTMLElement;
		const half = await row("Episode one");
		expect(within(half).getByText("15 min left")).toBeVisible();
		expect(
			within(half).getByTestId("listen-progress").firstElementChild,
		).toHaveStyle({ width: "75%" });
		const done = await row("Done");
		expect(within(done).getByText("Played")).toBeVisible();
		expect(
			within(done).getByTestId("listen-progress").firstElementChild,
		).toHaveStyle({ width: "100%" });
		const unknown = await row("Unknown length");
		expect(within(unknown).queryByTestId("listen-progress")).toBeNull();
		expect(within(unknown).queryByText(/min left/)).toBeNull();
		expect(
			within(await row("Never played")).queryByTestId("listen-progress"),
		).toBeNull();
	});

	/** The server keeps the record that the requests change. */
	function menuSetup(listen: FeedItem["listen"]) {
		let current = listen;
		return setup({
			more: {
				"GET /api/feed-items": () => ({
					body: { items: [{ ...episode, listen: current }], nextCursor: null },
				}),
				"PUT /api/feed-items/e1/listen": ({ body }) => {
					const { finished, ...rest } = body as {
						positionSeconds: number;
						finished?: boolean;
					};
					current = {
						...savedListen,
						...rest,
						finishedAt: finished ? "2026-11-08T11:00:00.000Z" : null,
					};
					return { body: { ...episode, listen: current } };
				},
				"DELETE /api/feed-items/e1/listen": () => {
					current = null;
					return { status: 204 };
				},
			},
		});
	}

	async function openMenu(user: ReturnType<typeof setup>["user"]) {
		await user.click(
			await screen.findByLabelText("Episode actions for Episode one"),
		);
		return screen.findByRole("menu");
	}

	it("marks an episode it never played as played", async () => {
		const { api, user } = menuSetup(null);
		const menu = await openMenu(user);
		expect(
			within(menu)
				.getAllByRole("menuitem")
				.map((m) => m.textContent),
		).toEqual(["Mark as played"]);
		await user.click(
			within(menu).getByRole("menuitem", { name: "Mark as played" }),
		);
		await waitFor(() =>
			expect(api.callsTo("PUT", "/api/feed-items/e1/listen")[0]?.body).toEqual({
				positionSeconds: 0,
				finished: true,
			}),
		);
		expect(await screen.findByText("Played")).toBeVisible();
	});

	it("marks an episode in progress as unplayed or removes it", async () => {
		const { api, user } = menuSetup(at(600));
		let menu = await openMenu(user);
		expect(
			within(menu)
				.getAllByRole("menuitem")
				.map((m) => m.textContent),
		).toEqual(["Mark as played", "Mark as unplayed", "Remove from history"]);
		await user.click(
			within(menu).getByRole("menuitem", { name: "Mark as unplayed" }),
		);
		await waitFor(() =>
			expect(api.callsTo("PUT", "/api/feed-items/e1/listen")[0]?.body).toEqual({
				positionSeconds: 0,
			}),
		);
		menu = await openMenu(user);
		await user.click(
			within(menu).getByRole("menuitem", { name: "Remove from history" }),
		);
		await waitFor(() =>
			expect(api.callsTo("DELETE", "/api/feed-items/e1/listen")).toHaveLength(
				1,
			),
		);
	});

	it("offers no Mark as played for a finished episode", async () => {
		const { user } = menuSetup(
			at(0, { finishedAt: "2026-11-08T10:00:00.000Z" }),
		);
		const menu = await openMenu(user);
		expect(
			within(menu)
				.getAllByRole("menuitem")
				.map((m) => m.textContent),
		).toEqual(["Mark as unplayed", "Remove from history"]);
	});

	it("stops the player before marking the loaded episode as played", async () => {
		const audio = stubAudio();
		const { api, user } = menuSetup(at(600));
		await user.click(
			await screen.findByRole("button", { name: "Play Episode one" }),
		);
		fireEvent.playing(audio.audio());
		audio.audio().currentTime = 900;
		const menu = await openMenu(user);
		await user.click(
			within(menu).getByRole("menuitem", { name: "Mark as played" }),
		);
		await waitFor(() =>
			expect(
				api.callsTo("PUT", "/api/feed-items/e1/listen").at(-1)?.body,
			).toEqual({
				positionSeconds: 0,
				finished: true,
			}),
		);
		expect(
			screen.queryByRole("region", { name: "Media player" }),
		).not.toBeInTheDocument();
		const bodies = api
			.callsTo("PUT", "/api/feed-items/e1/listen")
			.map((c) => c.body);
		expect(bodies.at(-2)).toEqual({ positionSeconds: 900 });
	});
});

describe("playing an episode", () => {
	function ready(listen: FeedItem["listen"] = null) {
		const audio = stubAudio();
		const app = setup({
			items: { latest: [{ ...episode, listen }] },
			more: {
				"PUT /api/feed-items/e1/listen": ({ body }) => ({
					body: { ...episode, listen: { ...savedListen, ...(body as object) } },
				}),
			},
		});
		const saves = () =>
			app.api.callsTo("PUT", "/api/feed-items/e1/listen").map((c) => c.body);
		const start = async () => {
			await app.user.click(
				await screen.findByRole("button", {
					name: /(Play|Pause) Episode one/,
				}),
			);
		};
		return { ...audio, ...app, saves, start };
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
		const { audio, start } = ready({ ...savedListen, positionSeconds: 1500 });
		await start();
		fireEvent.loadedMetadata(audio());
		expect(audio().currentTime).toBe(1500);
	});

	it("starts a finished episode at the beginning", async () => {
		const { audio, start } = ready({
			...savedListen,
			finishedAt: "2026-11-08T10:00:00.000Z",
		});
		await start();
		fireEvent.loadedMetadata(audio());
		expect(audio().currentTime).toBe(0);
	});

	it("saves the position when it is closed", async () => {
		const { audio, saves, start, user } = ready();
		await start();
		audio().currentTime = 1200;
		await user.click(
			within(bar()).getByRole("button", { name: "Close player" }),
		);
		await waitFor(() =>
			expect(saves().at(-1)).toEqual({ positionSeconds: 1200 }),
		);
		expect(localStorage.getItem("boardly.player.positions")).toBeNull();
	});

	it("finishes without an error and starts over from the beginning", async () => {
		const { audio, saves, start, user } = ready({
			...savedListen,
			positionSeconds: 1500,
		});
		await start();
		fireEvent.loadedMetadata(audio());
		audio().currentTime = 3599;
		fireEvent.ended(audio());
		expect(within(bar()).getByText("Finished")).toBeVisible();
		expect(within(bar()).queryByRole("alert")).not.toBeInTheDocument();
		await waitFor(() =>
			expect(saves().at(-1)).toEqual({ positionSeconds: 0, finished: true }),
		);
		await user.click(within(bar()).getByRole("button", { name: "Play" }));
		expect(within(bar()).getByText("Connecting…")).toBeVisible();
		// A browser starts a newly loaded source at 0; jsdom's stub doesn't.
		audio().currentTime = 0;
		fireEvent.loadedMetadata(audio());
		expect(audio().currentTime).toBe(0);
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
