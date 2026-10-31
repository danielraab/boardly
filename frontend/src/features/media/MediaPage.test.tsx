import {
	act,
	fireEvent,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
	apiError,
	mockApi,
	type Routes,
	signedIn,
	signedOut,
} from "#/test/api";
import { station, stubAudio } from "#/test/media";
import { renderApp } from "#/test/renderApp";
import type { MediaSource } from "./api";
import { resetRadioBrowser } from "./radioBrowser";

afterEach(() => {
	vi.restoreAllMocks();
	resetRadioBrowser();
});

const fm4 = station("fm4", "FM4", 0, {
	url: "https://orf-live.example/fm4-q2a",
	icon: { type: "symbol", name: "music" },
});
const jazz = station("jazz", "Jazz Radio", 1);

const directory = {
	"GET /json/servers": { body: [{ name: "de1.api.radio-browser.info" }] },
	"GET /json/stations/search": {
		body: [
			{
				stationuuid: "r1",
				name: "FM4",
				url: "https://orf-live.example/fm4-q2a/",
				favicon: "https://fm4.example/logo.png",
				country: "Austria",
				tags: "alternative",
				bitrate: 128,
			},
			{
				stationuuid: "r2",
				name: "FM4 Unlimited",
				url: "https://fm4.example/unlimited.mp3",
				url_resolved: "https://fm4.example/unlimited.mp3",
				favicon: "",
				country: "Austria",
				tags: "",
				bitrate: 0,
			},
			{
				stationuuid: "r3",
				name: "FM4 Old",
				url: "http://old.example:8000/fm4",
				favicon: "",
				country: "Austria",
				tags: "",
				bitrate: 64,
			},
		],
	},
};

/** A media API over a list that create, update, reorder and delete change. */
function fakeServer(initial: MediaSource[], more: Routes = {}) {
	let list = [...initial];
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/media": () => ({ body: list }),
		"POST /api/media": ({ body }) => {
			const b = body as { title: string; url: string };
			const created = station(`new${list.length}`, b.title, list.length, {
				url: b.url,
			});
			list = [...list, created];
			return { status: 201, body: created };
		},
		"PUT /api/media/order": ({ body }) => {
			const ids = body as string[];
			list = ids.map((id, position) => ({
				...(list.find((s) => s.id === id) as MediaSource),
				position,
			}));
			return { status: 204 };
		},
		...Object.fromEntries(
			initial.flatMap((s) => [
				[
					`PATCH /api/media/${s.id}`,
					({ body }: { body: unknown }) => {
						const patched = { ...s, ...(body as object) } as MediaSource;
						list = list.map((x) => (x.id === s.id ? patched : x));
						return { body: patched };
					},
				],
				[
					`DELETE /api/media/${s.id}`,
					() => {
						list = list.filter((x) => x.id !== s.id);
						return { status: 204 };
					},
				],
			]),
		),
		...directory,
		...more,
	});
	return { api, titles: () => list.map((s) => s.title) };
}

const rowTitles = () =>
	within(screen.getByRole("list"))
		.getAllByRole("listitem")
		.map((li) => li.querySelector(".font-medium")?.textContent);

async function openAdd(user: ReturnType<typeof renderApp>["user"]) {
	await user.click(await screen.findByRole("button", { name: "Add station" }));
	return screen.findByRole("dialog", { name: "Add station" });
}

describe("the media page", () => {
	it("lists the stations in their order with their host", async () => {
		fakeServer([fm4, jazz]);
		renderApp("/media");
		expect(await screen.findByRole("heading", { name: "Media" })).toBeVisible();
		await screen.findByText("Jazz Radio");
		expect(rowTitles()).toEqual(["FM4", "Jazz Radio"]);
		expect(screen.getByText("orf-live.example")).toBeVisible();
		expect(document.title).toBe("Media - Boardly");
		const [first, second] = within(screen.getByRole("list")).getAllByRole(
			"listitem",
		);
		expect(first.querySelector("[data-icon]")).toHaveAttribute(
			"data-icon",
			"music",
		);
		expect(second.querySelector("[data-icon]")).toHaveAttribute(
			"data-icon",
			"fallback",
		);
	});

	it("says when there are no stations", async () => {
		fakeServer([]);
		renderApp("/media");
		expect(await screen.findByText("No stations yet")).toBeVisible();
	});

	it("sends signed-out visitors to the login", async () => {
		mockApi({ "GET /api/me": signedOut });
		const { router } = renderApp("/media");
		await waitFor(() => expect(router.state.location.pathname).toBe("/login"));
	});

	it("renames a station and changes its icon", async () => {
		const { api } = fakeServer([fm4, jazz]);
		const { user } = renderApp("/media");
		await user.click(
			await screen.findByRole("button", { name: "Actions for FM4" }),
		);
		await user.click(await screen.findByRole("menuitem", { name: "Edit" }));
		const dialog = await screen.findByRole("dialog", { name: "Edit station" });
		const name = within(dialog).getByLabelText("Name");
		await user.clear(name);
		await user.type(name, "FM4 Wien");
		await user.click(within(dialog).getByRole("button", { name: "Save" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(rowTitles()).toEqual(["FM4 Wien", "Jazz Radio"]);
		expect(api.callsTo("PATCH", "/api/media/fm4")[0].body).toEqual({
			title: "FM4 Wien",
			url: fm4.url,
			icon: { type: "symbol", name: "music" },
		});
	});

	it("deletes the playing station after confirmation and stops the player", async () => {
		const audio = stubAudio();
		const server = fakeServer([fm4, jazz]);
		const { user } = renderApp("/media");
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		fireEvent.playing(audio.audio());
		expect(screen.getByRole("region", { name: "Media player" })).toBeVisible();
		await user.click(screen.getByRole("button", { name: "Actions for FM4" }));
		await user.click(await screen.findByRole("menuitem", { name: "Delete" }));
		const confirm = await screen.findByRole("alertdialog", {
			name: "Delete station?",
		});
		await user.click(within(confirm).getByRole("button", { name: "Delete" }));
		await waitFor(() => expect(rowTitles()).toEqual(["Jazz Radio"]));
		expect(screen.queryByRole("region", { name: "Media player" })).toBeNull();
		expect(server.titles()).toEqual(["Jazz Radio"]);
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
		const server = fakeServer([fm4, jazz]);
		const { user } = renderApp("/media");
		const handle = await screen.findByRole("button", {
			name: "Drag Jazz Radio",
		});
		act(() => handle.focus());
		await user.keyboard(" ");
		await user.keyboard("{ArrowUp}");
		await user.keyboard(" ");
		await waitFor(() => expect(server.titles()).toEqual(["Jazz Radio", "FM4"]));
		expect(server.api.callsTo("PUT", "/api/media/order")[0].body).toEqual([
			"jazz",
			"fm4",
		]);
		await waitFor(() => expect(rowTitles()).toEqual(["Jazz Radio", "FM4"]));
	});
});

describe("adding a station", () => {
	it("shows the directory notice and sends nothing before typing", async () => {
		const { api } = fakeServer([]);
		const { user } = renderApp("/media");
		const dialog = await openAdd(user);
		expect(
			within(dialog).getByText(
				/a free community directory\. Your search terms are sent to it\./,
			),
		).toBeVisible();
		const link = within(dialog).getByRole("link", {
			name: "radio-browser.info (opens in new tab)",
		});
		expect(link).toHaveAttribute("href", "https://www.radio-browser.info");
		expect(link).toHaveAttribute("target", "_blank");
		expect(link).toHaveAttribute("rel", "noopener noreferrer");
		await new Promise((r) => setTimeout(r, 400));
		expect(api.calls.filter((c) => c.path.startsWith("/json/"))).toEqual([]);
	});

	it("adds a playable search result with its logo", async () => {
		const server = fakeServer([jazz]);
		const { user } = renderApp("/media");
		const dialog = await openAdd(user);
		await user.type(within(dialog).getByLabelText("Search stations"), "fm4");
		await user.click(
			await within(dialog).findByRole("button", { name: "Add FM4" }),
		);
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(server.api.callsTo("POST", "/api/media")[0].body).toEqual({
			kind: "stream",
			title: "FM4",
			url: "https://orf-live.example/fm4-q2a/",
			logoUrl: "https://fm4.example/logo.png",
		});
		await waitFor(() => expect(rowTitles()).toEqual(["Jazz Radio", "FM4"]));
	});

	it("marks stations without https and those already added", async () => {
		fakeServer([fm4]);
		const { user } = renderApp("/media");
		const dialog = await openAdd(user);
		await user.type(within(dialog).getByLabelText("Search stations"), "fm4");
		const old = (await within(dialog).findByText("FM4 Old")).closest("li");
		expect(old).toHaveClass("opacity-60");
		expect(
			within(old as HTMLElement).getByText("Not playable (no HTTPS stream)"),
		).toBeVisible();
		expect(within(old as HTMLElement).queryByRole("button")).toBeNull();
		const added = within(dialog)
			.getByText("FM4", { exact: true })
			.closest("li");
		expect(within(added as HTMLElement).getByText("Added")).toBeVisible();
		expect(within(added as HTMLElement).queryByRole("button")).toBeNull();
		expect(
			within(dialog).getByRole("button", { name: "Add FM4 Unlimited" }),
		).toBeEnabled();
	});

	it("says when the directory can't be reached", async () => {
		fakeServer([], { "GET /json/stations/search": { status: 502 } });
		const { user } = renderApp("/media");
		const dialog = await openAdd(user);
		await user.type(within(dialog).getByLabelText("Search stations"), "fm4");
		expect(
			await within(dialog).findByText(
				"The station directory isn't reachable right now. You can still enter a URL.",
			),
		).toBeVisible();
		await user.click(within(dialog).getByRole("tab", { name: "Enter URL" }));
		expect(within(dialog).getByLabelText("Stream URL")).toBeVisible();
	});

	it("adds a station by URL", async () => {
		const server = fakeServer([]);
		const { user } = renderApp("/media");
		const dialog = await openAdd(user);
		await user.click(within(dialog).getByRole("tab", { name: "Enter URL" }));
		expect(
			within(dialog).getByText(/Only HTTPS streams can be played/),
		).toBeVisible();
		await user.type(within(dialog).getByLabelText("Name"), "My station");
		await user.type(
			within(dialog).getByLabelText("Stream URL"),
			"https://stream.example/live.mp3",
		);
		await user.click(within(dialog).getByRole("button", { name: "Add" }));
		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		expect(server.api.callsTo("POST", "/api/media")[0].body).toEqual({
			kind: "stream",
			title: "My station",
			url: "https://stream.example/live.mp3",
		});
		await waitFor(() => expect(rowTitles()).toEqual(["My station"]));
	});

	it("keeps the dialog open with a message when the stream isn't https", async () => {
		fakeServer([], {
			"POST /api/media": apiError(400, "stream_not_https"),
		});
		const { user } = renderApp("/media");
		const dialog = await openAdd(user);
		await user.click(within(dialog).getByRole("tab", { name: "Enter URL" }));
		await user.type(within(dialog).getByLabelText("Name"), "Old");
		await user.type(
			within(dialog).getByLabelText("Stream URL"),
			"http://old.example:8000/stream",
		);
		await user.click(within(dialog).getByRole("button", { name: "Add" }));
		expect(
			await within(dialog).findByText(
				"This station only offers an unencrypted stream, which Boardly can't play.",
			),
		).toBeVisible();
		expect(within(dialog).getByLabelText("Name")).toHaveValue("Old");
	});
});
