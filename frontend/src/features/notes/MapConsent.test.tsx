import { screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fakeNotesServer, note } from "#/test/notes";
import { renderApp } from "#/test/renderApp";
import type { Block, LocationBlock } from "./document/types";
import { resetMapConsent } from "./locations/mapConsent";

// jsdom has no WebGL, so the map is a stub. map.loaded counts how often its
// module was loaded, which is what fetches MapLibre and the style.
const map = vi.hoisted(() => ({ loaded: 0 }));
vi.mock("./locations/LocationMap", () => {
	map.loaded++;
	return {
		default: ({ point }: { point: { lat: number; lng: number } | null }) => (
			<div data-testid="map">
				{point ? `pin at ${point.lat}, ${point.lng}` : "no pin"}
			</div>
		),
	};
});

const key = "boardly.maps.allowedHosts";

const stephansplatz: LocationBlock = {
	type: "location",
	lat: 48.208176,
	lng: 16.373819,
	label: "Stephansplatz",
};
const text = (t: string): Block => ({
	type: "paragraph",
	content: [{ type: "text", text: t }],
});

beforeEach(() => resetMapConsent());

async function openMap(blocks: Block[] = [stephansplatz, text("Meet here")]) {
	const server = fakeNotesServer([
		note("n1", { title: "Trip", content: { v: 1, blocks } }),
	]);
	const app = renderApp("/notes/n1");
	await screen.findByRole("textbox", { name: "Content" });
	await app.user.click(
		screen.getByRole("button", { name: "Location: Stephansplatz" }),
	);
	const view = await screen.findByRole("dialog", { name: "Stephansplatz" });
	return { ...server, ...app, view };
}

describe("map consent", () => {
	// First in this file, so that nothing has loaded the map yet.
	it("asks before loading a map and loads nothing before", async () => {
		const { view } = await openMap();
		expect(view).toHaveTextContent(
			"The map is loaded from tiles.openfreemap.org, which sees your IP address and the area shown.",
		);
		expect(view).toHaveTextContent("48.208176, 16.373819");
		expect(within(view).getByRole("checkbox")).not.toBeChecked();
		expect(within(view).queryByTestId("map")).toBeNull();
		// Give a stray dynamic import the chance to run.
		await new Promise((r) => setTimeout(r, 50));
		expect(map.loaded).toBe(0);
	});

	it("loads once for the page, also in the form, without storing", async () => {
		const { user, view } = await openMap();
		await user.click(within(view).getByRole("button", { name: "Load map" }));
		expect(await within(view).findByTestId("map")).toHaveTextContent(
			"pin at 48.208176, 16.373819",
		);
		expect(
			within(view).queryByRole("button", {
				name: "Don't load maps automatically",
			}),
		).toBeNull();
		await user.click(within(view).getByRole("button", { name: "Edit" }));
		const form = await screen.findByRole("dialog", { name: "Edit location" });
		expect(await within(form).findByTestId("map")).toBeInTheDocument();
		expect(localStorage.getItem(key)).toBeNull();
	});

	it("stores consent with Always and withdraws it below the map", async () => {
		const first = await openMap();
		await first.user.click(
			within(first.view).getByRole("checkbox", {
				name: "Always load maps in this browser",
			}),
		);
		await first.user.click(
			within(first.view).getByRole("button", { name: "Load map" }),
		);
		expect(JSON.parse(localStorage.getItem(key) ?? "")).toEqual([
			"tiles.openfreemap.org",
		]);
		first.unmount();

		// As after a reload.
		resetMapConsent();
		const { user, view } = await openMap();
		expect(await within(view).findByTestId("map")).toBeInTheDocument();
		await user.click(
			within(view).getByRole("button", {
				name: "Don't load maps automatically",
			}),
		);
		expect(within(view).getByRole("status")).toHaveTextContent(
			"Maps will ask again.",
		);
		expect(within(view).getByTestId("map")).toBeInTheDocument();
		expect(localStorage.getItem(key)).toBeNull();
	});

	it("asks again for another host", async () => {
		localStorage.setItem(key, JSON.stringify(["maps.example.org"]));
		const { view } = await openMap();
		expect(
			within(view).getByRole("button", { name: "Load map" }),
		).toBeInTheDocument();
	});

	it("adds a location from the fields alone without loading the map", async () => {
		const server = fakeNotesServer([
			note("n1", { title: "Trip", content: { v: 1, blocks: [text("Trip")] } }),
		]);
		const { user } = renderApp("/notes/n1");
		await screen.findByRole("textbox", { name: "Content" });
		await user.click(screen.getByRole("button", { name: "Add location" }));
		const form = await screen.findByRole("dialog", { name: "Add location" });
		expect(
			within(form).getByRole("button", { name: "Load map" }),
		).toBeInTheDocument();
		await user.type(within(form).getByLabelText("Latitude"), "48.208176");
		await user.type(within(form).getByLabelText("Longitude"), "16.373819");
		await user.click(within(form).getByRole("button", { name: "Save" }));
		await waitFor(() => expect(server.notes[0].version).toBe(2), {
			timeout: 3000,
		});
		expect(server.notes[0].content.blocks).toEqual([
			text("Trip"),
			{ type: "location", lat: 48.208176, lng: 16.373819 },
		]);
		expect(within(form).queryByTestId("map")).toBeNull();
	});
});
