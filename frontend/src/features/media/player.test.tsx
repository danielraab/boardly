import {
	act,
	fireEvent,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { mockApi, signedIn } from "#/test/api";
import { station, stubAudio } from "#/test/media";
import { renderApp } from "#/test/renderApp";

const hls = vi.hoisted(() => ({ created: 0, sources: [] as string[] }));

vi.mock("hls.js", () => {
	class FakeHls {
		static isSupported = () => true;
		static Events = { ERROR: "hlsError" };
		constructor() {
			hls.created++;
		}
		on() {}
		loadSource(url: string) {
			hls.sources.push(url);
		}
		attachMedia() {}
		destroy() {}
	}
	return { default: FakeHls };
});

const fm4 = station("fm4", "FM4", 0);
const jazz = station("jazz", "Jazz Radio", 1);
const live = station("live", "HLS Radio", 2, {
	url: "https://hls.example/live/master.m3u8",
});

afterEach(() => {
	vi.restoreAllMocks();
	hls.created = 0;
	hls.sources = [];
});

function setup(routes = {}) {
	const audio = stubAudio();
	const api = mockApi({
		"GET /api/me": signedIn(),
		"GET /api/media": { body: [fm4, jazz, live] },
		...routes,
	});
	const app = renderApp("/media");
	return { ...app, ...audio, api };
}

const bar = () => screen.getByRole("region", { name: "Media player" });

describe("the media player", () => {
	it("plays a station from the media page in one audio element and switches stations", async () => {
		const { user, audio, play } = setup();
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		expect(audio().src).toBe(fm4.url);
		expect(play).toHaveBeenCalledTimes(1);
		expect(within(bar()).getByText("FM4")).toBeVisible();
		expect(within(bar()).getByText("Connecting…")).toBeVisible();
		fireEvent.playing(audio());
		expect(within(bar()).getByText("Live")).toBeVisible();
		expect(screen.getByRole("button", { name: "Pause FM4" })).toHaveAttribute(
			"aria-pressed",
			"true",
		);

		await user.click(screen.getByRole("button", { name: "Play Jazz Radio" }));
		expect(audio().src).toBe(jazz.url);
		expect(within(bar()).getByText("Jazz Radio")).toBeVisible();
		expect(screen.getByRole("button", { name: "Play FM4" })).toBeVisible();
	});

	it("pauses and resumes from the live position", async () => {
		const { user, audio, play, pause } = setup();
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		fireEvent.playing(audio());
		await user.click(within(bar()).getByRole("button", { name: "Pause" }));
		expect(pause).toHaveBeenCalled();
		expect(within(bar()).getByText("Paused")).toBeVisible();
		const loads = vi.mocked(audio().load).mock.calls.length;
		await user.click(within(bar()).getByRole("button", { name: "Play" }));
		expect(play).toHaveBeenCalledTimes(2);
		// The stream is loaded again rather than resumed from its buffer.
		expect(vi.mocked(audio().load).mock.calls.length).toBeGreaterThan(loads);
		expect(audio().src).toBe(fm4.url);
	});

	it("closes and stops", async () => {
		const { user, audio } = setup();
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		fireEvent.playing(audio());
		await user.click(screen.getByRole("button", { name: "Close player" }));
		expect(screen.queryByRole("region", { name: "Media player" })).toBeNull();
		expect(audio().getAttribute("src")).toBeNull();
	});

	it("shows an error and tries again", async () => {
		const { user, audio, play } = setup();
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		fireEvent.error(audio());
		expect(
			within(bar()).getByText("This station can't be played right now"),
		).toBeVisible();
		await user.click(within(bar()).getByRole("button", { name: "Try again" }));
		expect(play).toHaveBeenCalledTimes(2);
		fireEvent.playing(audio());
		expect(within(bar()).getByText("Live")).toBeVisible();
	});

	it("shows an error when the browser refuses to play", async () => {
		const { user, state } = setup();
		state.playResult = () =>
			Promise.reject(new DOMException("unsupported", "NotSupportedError"));
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		expect(
			await within(bar()).findByText("This station can't be played right now"),
		).toBeVisible();
	});

	it("remembers the volume and the muted state", async () => {
		const first = setup();
		await first.user.click(
			await screen.findByRole("button", { name: "Play FM4" }),
		);
		fireEvent.change(within(bar()).getByRole("slider", { name: "Volume" }), {
			target: { value: "40" },
		});
		await first.user.click(within(bar()).getByRole("button", { name: "Mute" }));
		expect(first.audio().volume).toBeCloseTo(0.4);
		expect(first.audio().muted).toBe(true);
		first.unmount();

		const second = setup();
		await second.user.click(
			await screen.findByRole("button", { name: "Play FM4" }),
		);
		expect(within(bar()).getByRole("slider", { name: "Volume" })).toHaveValue(
			"40",
		);
		expect(second.audio().volume).toBeCloseTo(0.4);
		expect(within(bar()).getByRole("button", { name: "Unmute" })).toBeVisible();
	});

	it("works with full volume when the browser can't store it", async () => {
		vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => {
			throw new Error("blocked");
		});
		const { user, audio } = setup();
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		expect(audio().volume).toBe(1);
		fireEvent.change(within(bar()).getByRole("slider", { name: "Volume" }), {
			target: { value: "20" },
		});
		expect(audio().volume).toBeCloseTo(0.2);
	});

	it("keeps playing while moving to another page", async () => {
		const { user, audio, pause, router } = setup({
			"GET /api/notes": { body: [] },
			"GET /api/note-categories": { body: [] },
		});
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		fireEvent.playing(audio());
		const element = audio();
		const pauses = pause.mock.calls.length;
		await act(() => router.navigate({ to: "/notes" }));
		await waitFor(() => expect(router.state.location.pathname).toBe("/notes"));
		expect(audio()).toBe(element);
		expect(element.src).toBe(fm4.url);
		expect(pause.mock.calls.length).toBe(pauses);
		expect(within(bar()).getByText("FM4")).toBeVisible();
	});

	it("stops when the user signs out", async () => {
		const { user, audio } = setup({
			"POST /api/auth/logout": { status: 204 },
		});
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		fireEvent.playing(audio());
		await user.click(screen.getByRole("button", { name: "User menu" }));
		await user.click(await screen.findByRole("menuitem", { name: "Sign out" }));
		await waitFor(() =>
			expect(screen.queryByRole("region", { name: "Media player" })).toBeNull(),
		);
		expect(audio().getAttribute("src")).toBeNull();
	});

	it("loads the HLS script only for HLS streams the browser can't play itself", async () => {
		const { user, audio } = setup();
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		expect(hls.created).toBe(0);
		await user.click(screen.getByRole("button", { name: "Play HLS Radio" }));
		await waitFor(() => expect(hls.sources).toEqual([live.url]));
		expect(audio().getAttribute("src")).toBeNull();
	});

	it("tells the operating system what plays and follows its controls", async () => {
		const handlers: Record<string, (() => void) | null> = {};
		const session = {
			metadata: null as unknown,
			setActionHandler: (action: string, handler: (() => void) | null) => {
				handlers[action] = handler;
			},
		};
		vi.stubGlobal("navigator", { ...navigator, mediaSession: session });
		vi.stubGlobal(
			"MediaMetadata",
			class {
				constructor(public init: Record<string, unknown>) {}
			},
		);
		const { user, audio, pause } = setup();
		await user.click(await screen.findByRole("button", { name: "Play FM4" }));
		fireEvent.playing(audio());
		expect(session.metadata).toMatchObject({
			init: { title: "FM4", artist: "Boardly" },
		});
		act(() => handlers.pause?.());
		expect(pause).toHaveBeenCalled();
		expect(within(bar()).getByText("Paused")).toBeVisible();
		act(() => handlers.stop?.());
		expect(screen.queryByRole("region", { name: "Media player" })).toBeNull();
		expect(handlers.play).toBeNull();
	});
});
