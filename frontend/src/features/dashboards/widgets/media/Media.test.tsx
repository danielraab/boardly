import {
	act,
	fireEvent,
	screen,
	waitFor,
	within,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ContentWidget, Widget } from "#/features/dashboards/api";
import type { PlayableSource } from "#/features/media/api";
import i18n from "#/i18n";
import { mockApi, type Routes, signedIn } from "#/test/api";
import { section, viewData } from "#/test/dashboards";
import { station, stubAudio } from "#/test/media";
import { renderApp } from "#/test/renderApp";
import type { MediaConfig } from ".";

afterEach(() => {
	vi.restoreAllMocks();
});

const fm4 = station("fm4", "FM4", 0, {
	icon: { type: "symbol", name: "music" },
});
const jazz = station("jazz", "Jazz Radio", 1);
const playable = ({ position: _, ...s }: typeof fm4): PlayableSource => s;

function mediaWidget(
	config: Partial<MediaConfig> = {},
	sources: PlayableSource[] | undefined = [playable(fm4), playable(jazz)],
): ContentWidget {
	return {
		id: "w1",
		kind: "media",
		width: 2,
		height: "auto",
		config: {
			sourceIds: ["fm4", "jazz"],
			title: null,
			layout: "list",
			size: "M",
			showIfPublic: false,
			...config,
		},
		data: sources && { sources },
	};
}

function dashboard(
	sections: () => Widget[],
	{ isOwner = true, library = [fm4, jazz], more = {} as Routes } = {},
) {
	return mockApi({
		"GET /api/me": isOwner
			? signedIn()
			: {
					status: 401,
					body: { error: { code: "unauthenticated", message: "" } },
				},
		"GET /api/media": { body: library },
		"GET /api/u/ann/home": () => ({
			body: viewData(
				sections(),
				isOwner ? {} : { isOwner: false, id: undefined },
			),
		}),
		...more,
	});
}

describe("the media widget", () => {
	it("plays a station and marks it, also in a second widget, and pauses it", async () => {
		const audio = stubAudio();
		dashboard(() => [
			section("Work", {}, mediaWidget({ title: "Radio" }), {
				...mediaWidget({ layout: "tiles" }, [playable(fm4)]),
				id: "w2",
			}),
		]);
		const { user } = renderApp("/ann/home");
		expect(await screen.findByRole("heading", { name: "Radio" })).toBeVisible();
		const [listButton, tileButton] = await screen.findAllByRole("button", {
			name: "Play FM4",
		});
		await user.click(listButton);
		expect(audio.audio().src).toBe(fm4.url);
		fireEvent.playing(audio.audio());
		const playing = screen.getAllByRole("button", { name: "Pause FM4" });
		expect(playing).toHaveLength(2);
		for (const b of playing) expect(b).toHaveAttribute("aria-pressed", "true");
		expect(tileButton).toHaveAttribute("data-state", "playing");

		await user.click(playing[1]);
		expect(audio.pause).toHaveBeenCalled();
		expect(
			screen.getAllByRole("button", { name: "Play FM4" })[0],
		).toHaveAttribute("data-state", "paused");
	});

	it("lets visitors play the stations of a public widget", async () => {
		const audio = stubAudio();
		dashboard(
			() => [section("Work", {}, mediaWidget({ showIfPublic: true }))],
			{
				isOwner: false,
			},
		);
		const { user } = renderApp("/ann/home");
		await user.click(
			await screen.findByRole("button", { name: "Play Jazz Radio" }),
		);
		expect(audio.audio().src).toBe(jazz.url);
		expect(screen.getByRole("region", { name: "Media player" })).toBeVisible();
	});

	it("says when none of its stations exist", async () => {
		dashboard(() => [section("Work", {}, mediaWidget({}, []))]);
		renderApp("/ann/home");
		expect(
			await screen.findByText("These stations no longer exist"),
		).toBeVisible();
	});

	it("shows the current library in edit mode and doesn't play there", async () => {
		const audio = stubAudio();
		dashboard(
			() => [
				section(
					"Work",
					{},
					mediaWidget({ sourceIds: ["jazz", "gone", "fm4"] }, []),
				),
			],
			{ library: [{ ...fm4, title: "FM4 Wien" }, jazz] },
		);
		const { user } = renderApp("/ann/home?edit=true");
		const jazzButton = await screen.findByRole("button", {
			name: "Play Jazz Radio",
		});
		expect(screen.getByRole("button", { name: "Play FM4 Wien" })).toBeVisible();
		await user.click(jazzButton);
		expect(audio.play).not.toHaveBeenCalled();
		expect(screen.queryByRole("region", { name: "Media player" })).toBeNull();
	});

	it("is added through the picker with its defaults and saves its settings", async () => {
		const api = dashboard(() => [section("Work")], {
			more: {
				"PUT /api/dashboards/d1/layout": ({ body }) => ({
					body: {
						version: 4,
						sections: (body as { sections: Widget[] }).sections,
					},
				}),
			},
		});
		const { user, router } = renderApp("/ann/home?edit=true");
		await user.click(await screen.findByRole("button", { name: "Add widget" }));
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		await user.click(within(picker).getByRole("button", { name: /^Media/ }));

		const dialog = await screen.findByRole("dialog", { name: "Add media" });
		const add = within(dialog).getByRole("button", { name: "Add" });
		expect(add).toBeDisabled();
		expect(within(dialog).getByLabelText("Layout")).toHaveValue("list");
		expect(within(dialog).getByLabelText("Size")).toHaveValue("M");
		const pub = within(dialog).getByLabelText("Show on public dashboard");
		expect(pub).not.toBeChecked();
		expect(
			within(dialog).getByText(
				/Visitors of a public dashboard can play these stations/,
			),
		).toBeVisible();

		const search = await within(dialog).findByLabelText("Stations");
		await user.type(search, "fm");
		await user.click(await screen.findByRole("option", { name: "FM4" }));
		await user.type(search, "jazz");
		await user.click(await screen.findByRole("option", { name: "Jazz Radio" }));
		expect(add).toBeEnabled();
		await user.click(
			within(dialog).getByRole("button", { name: "Move Jazz Radio up" }),
		);
		await user.selectOptions(within(dialog).getByLabelText("Layout"), "tiles");
		await user.click(pub);
		await user.click(add);

		await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
		await user.click(screen.getByRole("button", { name: "Save and exit" }));
		await waitFor(() => expect(router.state.location.search).toEqual({}));
		const [call] = api.callsTo("PUT", "/api/dashboards/d1/layout");
		expect((call.body as { sections: Widget[] }).sections[0]).toMatchObject({
			children: [
				{
					kind: "media",
					config: {
						sourceIds: ["jazz", "fm4"],
						title: null,
						layout: "tiles",
						size: "M",
						showIfPublic: true,
					},
				},
			],
		});
	});

	it("points to the media page when the library is empty", async () => {
		dashboard(() => [section("Work")], { library: [] });
		const { user } = renderApp("/ann/home?edit=true");
		await user.click(await screen.findByRole("button", { name: "Add widget" }));
		const picker = await screen.findByRole("dialog", { name: "Add widget" });
		await user.click(within(picker).getByRole("button", { name: /^Media/ }));
		const dialog = await screen.findByRole("dialog", { name: "Add media" });
		expect(
			within(dialog).getByText(/Your media library is empty\./),
		).toBeVisible();
		expect(
			within(dialog).getByRole("link", { name: "Add stations" }),
		).toHaveAttribute("href", "/media");
	});

	it("shows its picker entry in German", async () => {
		dashboard(() => [section("Work")]);
		const { user } = renderApp("/ann/home?edit=true");
		await screen.findByRole("button", { name: "Add widget" });
		await act(() => i18n.changeLanguage("de"));
		await user.click(screen.getByRole("button", { name: "Widget hinzufügen" }));
		const picker = await screen.findByRole("dialog", {
			name: "Widget hinzufügen",
		});
		expect(
			within(within(picker).getByRole("region", { name: "Medien" })).getByText(
				"Spielt Sender aus deiner Mediathek ab.",
			),
		).toBeVisible();
	});
});
