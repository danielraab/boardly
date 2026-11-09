import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { mockApi, type Routes, signedIn } from "#/test/api";
import { embed, station, stubAudio } from "#/test/media";
import { renderApp } from "#/test/renderApp";

const fm4 = station("fm4", "FM4", 0);
const album = embed("album", "Global Warming", 1);
const other = embed("other", "Second Album", 2);
const noticeKey = "boardly.embedNotice";

afterEach(() => {
	vi.restoreAllMocks();
	localStorage.removeItem(noticeKey);
});

function setup({ confirmed = true, routes = {} as Routes } = {}) {
	if (confirmed) localStorage.setItem(noticeKey, "spotify");
	const audio = stubAudio();
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/media": { body: [fm4, album, other] },
		...routes,
	});
	return { ...renderApp("/media"), ...audio, api };
}

const frame = () => document.querySelector("iframe");
const dock = () => screen.getByRole("region", { name: "Embedded player" });
const miniBar = () => screen.queryByRole("region", { name: "Media player" });
const openButton = (name: string) =>
	screen.findByRole("button", { name: `Open ${name}` });

describe("the embed dock", () => {
	it("shows the provider's player in a plain iframe, without a provider script", async () => {
		const { user } = setup();
		await user.click(await openButton("Global Warming"));

		const iframe = frame();
		expect(iframe).toHaveAttribute("src", album.embedUrl);
		expect(iframe).toHaveAttribute("title", "Global Warming");
		expect(iframe?.getAttribute("allow")).toContain("encrypted-media");
		expect(within(dock()).getByText("Global Warming")).toBeVisible();
		expect(document.querySelector('script[src*="spotify"]')).toBeNull();
		expect(await openButton("Global Warming")).toHaveAttribute(
			"aria-pressed",
			"true",
		);
	});

	it("floats on larger screens and spans the width on phones", async () => {
		const { user } = setup();
		await user.click(await openButton("Global Warming"));
		expect(dock().className).toContain("inset-x-0");
		expect(dock().className).toContain("sm:right-4");
		expect(dock().className).toContain("sm:w-[360px]");
		expect(frame()?.className).toContain("h-[152px]");
		expect(frame()?.className).toContain("sm:h-[352px]");
	});

	it("closes and removes the iframe", async () => {
		const { user } = setup();
		await user.click(await openButton("Global Warming"));
		await user.click(
			within(dock()).getByRole("button", { name: "Close player" }),
		);
		expect(frame()).toBeNull();
		expect(
			screen.queryByRole("region", { name: "Embedded player" }),
		).toBeNull();
	});

	it("doesn't reload the player when an open embed is activated again", async () => {
		const { user } = setup();
		await user.click(await openButton("Global Warming"));
		const iframe = frame();
		await user.click(await openButton("Global Warming"));
		expect(frame()).toBe(iframe);
		expect(iframe?.isConnected).toBe(true);
	});

	it("replaces a station, and a station replaces it", async () => {
		const { user, audio, pause } = setup();
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		fireEvent.playing(audio());
		expect(miniBar()).toBeVisible();

		await user.click(await openButton("Global Warming"));
		expect(pause).toHaveBeenCalled();
		expect(audio().getAttribute("src")).toBeNull();
		expect(miniBar()).toBeNull();
		expect(frame()).not.toBeNull();

		await user.click(screen.getByRole("button", { name: "Play FM4" }));
		expect(frame()).toBeNull();
		expect(audio().src).toBe(fm4.url);
		expect(miniBar()).toBeVisible();
	});

	it("switches from one embed to another", async () => {
		const { user } = setup();
		await user.click(await openButton("Global Warming"));
		await user.click(await openButton("Second Album"));
		expect(document.querySelectorAll("iframe")).toHaveLength(1);
		expect(frame()).toHaveAttribute("src", other.embedUrl);
	});

	async function rename(
		user: ReturnType<typeof setup>["user"],
		field: string,
		value: string,
	) {
		await user.click(
			screen.getByRole("button", { name: "Actions for Global Warming" }),
		);
		await user.click(await screen.findByRole("menuitem", { name: "Edit" }));
		const dialog = await screen.findByRole("dialog", { name: "Edit embed" });
		const input = within(dialog).getByLabelText(field);
		await user.clear(input);
		await user.type(input, value);
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
	}

	it("shows a new title without reloading the player", async () => {
		const { user } = setup({
			routes: {
				"PATCH /api/media/album": { body: { ...album, title: "Renamed" } },
			},
		});
		await user.click(await openButton("Global Warming"));
		const iframe = frame();
		await rename(user, "Name", "Renamed");
		expect(await within(dock()).findByText("Renamed")).toBeVisible();
		expect(frame()).toBe(iframe);
	});

	it("loads the new player when the link changed", async () => {
		const embedUrl = "https://open.spotify.com/embed/playlist/new";
		const { user } = setup({
			routes: {
				"PATCH /api/media/album": {
					body: {
						...album,
						url: "https://open.spotify.com/playlist/new",
						embedUrl,
					},
				},
			},
		});
		await user.click(await openButton("Global Warming"));
		const iframe = frame();
		await rename(user, "Link", "https://open.spotify.com/playlist/new");
		await waitFor(() => expect(frame()).toHaveAttribute("src", embedUrl));
		expect(iframe?.isConnected).toBe(false);
	});

	it("closes when the open embed is deleted", async () => {
		const { user } = setup({
			routes: { "DELETE /api/media/album": { status: 204 } },
		});
		await user.click(await openButton("Global Warming"));
		await user.click(
			screen.getByRole("button", { name: "Actions for Global Warming" }),
		);
		await user.click(await screen.findByRole("menuitem", { name: "Delete" }));
		const dialog = await screen.findByRole("alertdialog", {
			name: "Delete embed?",
		});
		await user.click(within(dialog).getByRole("button", { name: "Delete" }));
		await waitFor(() => expect(frame()).toBeNull());
	});
});

describe("the first-use notice for embeds", () => {
	const notice = () =>
		screen.findByRole("alertdialog", { name: "Open the Spotify player?" });

	it("asks before anything is loaded, then remembers the answer", async () => {
		const { user } = setup({ confirmed: false });
		await user.click(await openButton("Global Warming"));
		const dialog = await notice();
		expect(
			within(dialog).getByText(/connects your browser to Spotify/),
		).toBeVisible();
		expect(frame()).toBeNull();

		await user.click(
			within(dialog).getByRole("button", { name: "Open player" }),
		);
		expect(frame()).toHaveAttribute("src", album.embedUrl);
		expect(localStorage.getItem(noticeKey)).toBe("spotify");

		await user.click(await openButton("Second Album"));
		expect(screen.queryByRole("alertdialog")).toBeNull();
		expect(frame()).toHaveAttribute("src", other.embedUrl);
	});

	it("leaves everything as it is when cancelled", async () => {
		const { user, audio } = setup({ confirmed: false });
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		fireEvent.playing(audio());

		await user.click(await openButton("Global Warming"));
		const dialog = await notice();
		await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
		await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());

		expect(frame()).toBeNull();
		expect(audio().src).toBe(fm4.url);
		expect(miniBar()).toBeVisible();
		expect(localStorage.getItem(noticeKey)).toBeNull();
	});

	it("asks each time when the answer can't be stored", async () => {
		const { user } = setup({ confirmed: false });
		const setItem = vi
			.spyOn(Storage.prototype, "setItem")
			.mockImplementation(() => {
				throw new Error("denied");
			});
		await user.click(await openButton("Global Warming"));
		await user.click(
			within(await notice()).getByRole("button", { name: "Open player" }),
		);
		expect(setItem).toHaveBeenCalled();
		expect(frame()).toHaveAttribute("src", album.embedUrl);

		await user.click(await openButton("Second Album"));
		expect(await notice()).toBeVisible();
	});
});
